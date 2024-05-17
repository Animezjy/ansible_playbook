from flask import Flask, Response, request
from metrics import MetricsExporter

app = Flask(__name__)
metrics_exporter = MetricsExporter()


@app.route("/", methods=["GET"])
def index():
    metrics_exporter.increment_http_requests(request.method, request.path)
    return "Hello, this is the home page!"


@app.route("/metrics")
def metrics():
    return Response(metrics_exporter.get_metrics(), mimetype="text/plain")


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000)
