const express = require('express');
const { SNSClient, PublishCommand } = require('@aws-sdk/client-sns');
const { S3Client, PutObjectCommand, GetObjectCommand } = require('@aws-sdk/client-s3');
const { getSignedUrl } = require('@aws-sdk/s3-request-presigner');

const app = express();
app.use(express.json());

// Configuration
const PORT = process.env.PORT || 8082;
const AWS_ENDPOINT = process.env.AWS_ENDPOINT || 'http://localhost:4566';
const AWS_REGION = process.env.AWS_REGION || 'us-east-1';
const NOTIFICATIONS_TOPIC_ARN = process.env.NOTIFICATIONS_TOPIC_ARN || '';
const LOGS_BUCKET_URL = process.env.LOGS_BUCKET_URL || '';

// AWS clients
const snsClient = new SNSClient({
    endpoint: AWS_ENDPOINT,
    region: AWS_REGION,
    credentials: {
        accessKeyId: 'test',
        secretAccessKey: 'test'
    }
});

const s3Client = new S3Client({
    endpoint: AWS_ENDPOINT,
    region: AWS_REGION,
    credentials: {
        accessKeyId: 'test',
        secretAccessKey: 'test'
    },
    forcePathStyle: true
});

// In-memory notification log
const notifications = [];

// Health check
app.get('/health', (req, res) => {
    res.json({ status: 'healthy', service: 'notification-service' });
});

// Send notification
app.post('/notify', async (req, res) => {
    const { user_id, message, channel = 'email' } = req.body;

    const notification = {
        id: notifications.length + 1,
        user_id,
        message,
        channel,
        status: 'pending',
        created_at: new Date().toISOString()
    };

    try {
        // Publish to SNS
        if (NOTIFICATIONS_TOPIC_ARN) {
            await snsClient.send(new PublishCommand({
                TopicArn: NOTIFICATIONS_TOPIC_ARN,
                Message: JSON.stringify({
                    event: 'notification_sent',
                    notification,
                    timestamp: new Date().toISOString()
                }),
                Subject: `Notification for user ${user_id}`
            }));
            console.log(`Published notification ${notification.id} to SNS`);
        }

        notification.status = 'sent';
        notifications.push(notification);

        // Log to S3
        await logToS3(notification);

        res.status(201).json(notification);
    } catch (error) {
        console.error('Failed to send notification:', error);
        notification.status = 'failed';
        notifications.push(notification);
        res.status(500).json({ error: error.message });
    }
});

// Get all notifications
app.get('/notifications', (req, res) => {
    res.json(notifications);
});

// Get notification by ID
app.get('/notifications/:id', (req, res) => {
    const notification = notifications.find(n => n.id === parseInt(req.params.id));
    if (!notification) {
        return res.status(404).json({ error: 'Notification not found' });
    }
    res.json(notification);
});

// Generate presigned URL for log file
app.get('/logs/:key', async (req, res) => {
    try {
        const command = new GetObjectCommand({
            Bucket: 'notification-logs',
            Key: req.params.key
        });
        const url = await getSignedUrl(s3Client, command, { expiresIn: 3600 });
        res.json({ url });
    } catch (error) {
        console.error('Failed to generate presigned URL:', error);
        res.status(500).json({ error: error.message });
    }
});

// Helper: Log notification to S3
async function logToS3(notification) {
    try {
        const key = `${new Date().toISOString().split('T')[0]}/notification-${notification.id}.json`;
        await s3Client.send(new PutObjectCommand({
            Bucket: 'notification-logs',
            Key: key,
            Body: JSON.stringify(notification, null, 2),
            ContentType: 'application/json'
        }));
        console.log(`Logged notification ${notification.id} to S3: ${key}`);
    } catch (error) {
        console.error('Failed to log to S3:', error);
    }
}

app.listen(PORT, () => {
    console.log(`notification-service starting on port ${PORT}`);
});
