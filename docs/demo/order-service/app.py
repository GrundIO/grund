import os
import json
import time
import logging
from datetime import datetime
from flask import Flask, request, jsonify
import psycopg2
from psycopg2.extras import RealDictCursor
import boto3
import requests

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
DATABASE_URL = os.getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/orders_db")
USER_SERVICE_URL = os.getenv("USER_SERVICE_URL", "http://localhost:8080")
SQS_QUEUE_URL = os.getenv("ORDERS_QUEUE_URL", "")
AWS_ENDPOINT = os.getenv("AWS_ENDPOINT", "http://localhost:4566")
AWS_REGION = os.getenv("AWS_REGION", "us-east-1")

# Database connection with retry
def get_db():
    for i in range(30):
        try:
            conn = psycopg2.connect(DATABASE_URL, cursor_factory=RealDictCursor)
            return conn
        except Exception as e:
            logger.info(f"Waiting for database... ({i+1}/30)")
            time.sleep(1)
    raise Exception("Could not connect to database")

# Initialize database
def init_db():
    conn = get_db()
    cur = conn.cursor()
    cur.execute("""
        CREATE TABLE IF NOT EXISTS orders (
            id SERIAL PRIMARY KEY,
            user_id INTEGER NOT NULL,
            product VARCHAR(255) NOT NULL,
            quantity INTEGER NOT NULL,
            status VARCHAR(50) DEFAULT 'pending',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    conn.commit()
    cur.close()
    conn.close()
    logger.info("Database initialized")

# SQS client
def get_sqs():
    return boto3.client(
        'sqs',
        endpoint_url=AWS_ENDPOINT,
        region_name=AWS_REGION,
        aws_access_key_id='test',
        aws_secret_access_key='test'
    )

@app.route('/health')
def health():
    return jsonify({"status": "healthy", "service": "order-service"})

@app.route('/orders', methods=['GET', 'POST'])
def orders():
    if request.method == 'GET':
        conn = get_db()
        cur = conn.cursor()
        cur.execute("SELECT * FROM orders ORDER BY id")
        orders = cur.fetchall()
        cur.close()
        conn.close()
        return jsonify([dict(o) for o in orders])

    elif request.method == 'POST':
        data = request.json
        user_id = data.get('user_id')
        product = data.get('product')
        quantity = data.get('quantity', 1)

        # Verify user exists
        try:
            resp = requests.get(f"{USER_SERVICE_URL}/users/{user_id}", timeout=5)
            if resp.status_code == 404:
                return jsonify({"error": "User not found"}), 404
        except requests.RequestException as e:
            logger.warning(f"Could not verify user: {e}")

        # Create order
        conn = get_db()
        cur = conn.cursor()
        cur.execute(
            "INSERT INTO orders (user_id, product, quantity) VALUES (%s, %s, %s) RETURNING id, created_at",
            (user_id, product, quantity)
        )
        result = cur.fetchone()
        conn.commit()
        cur.close()
        conn.close()

        order = {
            "id": result['id'],
            "user_id": user_id,
            "product": product,
            "quantity": quantity,
            "status": "pending",
            "created_at": result['created_at'].isoformat()
        }

        # Send to SQS queue
        if SQS_QUEUE_URL:
            try:
                sqs = get_sqs()
                sqs.send_message(
                    QueueUrl=SQS_QUEUE_URL,
                    MessageBody=json.dumps({
                        "event": "order_created",
                        "order": order,
                        "timestamp": datetime.utcnow().isoformat()
                    })
                )
                logger.info(f"Order {order['id']} sent to queue")
            except Exception as e:
                logger.error(f"Failed to send to SQS: {e}")

        return jsonify(order), 201

@app.route('/orders/<int:order_id>')
def get_order(order_id):
    conn = get_db()
    cur = conn.cursor()
    cur.execute("SELECT * FROM orders WHERE id = %s", (order_id,))
    order = cur.fetchone()
    cur.close()
    conn.close()

    if not order:
        return jsonify({"error": "Order not found"}), 404
    return jsonify(dict(order))

if __name__ == '__main__':
    init_db()
    port = int(os.getenv("PORT", 8081))
    logger.info(f"order-service starting on port {port}")
    app.run(host='0.0.0.0', port=port)
