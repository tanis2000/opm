import requests
import json
import websocket
import base64
import threading
import logging

log = logging.getLogger(__name__)
handler = logging.StreamHandler()
formatter = logging.Formatter(
        '[%(asctime)s] [%(levelname)-8s] %(message)s')
handler.setFormatter(formatter)
log.addHandler(handler)
log.setLevel(logging.DEBUG)


def connect(addr, id):
    ws = websocket.create_connection(addr)
    session = requests.session()
    session.verify = True

    while True:
        raw = ws.recv()
        log.debug("#%03d <<< %d", id, len(raw))
        data = json.loads(raw)
        session.headers = {"User-Agent": data["user"]}

        if data["cont"] != "":
            session.headers.update({"Content-Type": data["cont"]})

        # Convert from base64 back to bytes
        payload = base64.b64decode(data["data"])

        if data["meth"] == "GET":
            response = session.get(data["host"], data=payload, allow_redirects=False)
        else:
            response = session.post(data["host"], data=payload, allow_redirects=False)
        content = response.content

        content = base64.b64encode(content)
        content = content.decode()

        location = ""
        if "Location" in response.headers.keys():
            location = response.headers["Location"]

        r = {"status": response.status_code, "response": content, "location": location}
        log.debug("#%03d >>> %d", id, len(response.content))
        ws.send(json.dumps(r))


def main():
    num = 200

    for i in range(num):
        t = threading.Thread(target=connect, args=["ws://localhost:8080/websocket", i])
        log.debug("Starting %d", i)
        t.start()


if __name__ == "__main__":
    main()
