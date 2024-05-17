from flask import Flask, Response, render_template

app = Flask(__name__)


@app.route("/")
def home():
    return Response("hfdffasafadfadfasahaah")


if __name__ == "__main__":
    app.run(debug=True)
