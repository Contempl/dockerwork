from flask import Flask
import os

app = Flask(__name__)

@app.route("/")
def hello():
    return f"Привет от {os.environ.get('APP_NAME', 'неизвестный')}"

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)