from prometheus_client import Counter, Gauge, generate_latest


class MetricsExporter:
    def __init__(self):
        self.http_requests_total = Counter(
            "http_requests_total", "Total HTTP Requests (count)", ["method", "endpoint"]
        )
        self.custom_gauge = Gauge("custom_gauge", "An example gauge metric")

    def increment_http_requests(self, method, endpoint):
        self.http_requests_total.labels(method, endpoint).inc()

    def set_custom_gauge(self, value):
        self.custom_gauge.set(value)

    @staticmethod
    def get_metrics():
        return generate_latest()
