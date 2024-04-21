# pip install rel
import rel

# pip install websocket-client
import websocket

# https://docs.tendermint.com/v0.34/rpc/
RPC_URL = "15.204.143.232:26657"
RPC_URL = f"ws://{RPC_URL}/websocket"

# For 443 / https connections, use wss
# RPC_URL = f"wss://rpc.juno.strange.love:443/websocket"

import json


def on_message(ws, message):
    msg = json.loads(message)

    if msg.get("result") == {}:
        print("Subscribed to New Block...")
        return

    # with open("/home/reecepbcups/Desktop/tm_sub.txt", "w") as f:
    #     f.write(json.dumps(msg["result"]))

    print(
        f"""New Block: {msg["result"]["data"]["value"]["block"]["header"]["height"]}"""
    )


def on_error(ws, error):
    print("error", error)


def on_close(ws, close_status_code, close_msg):
    print("### closed ###")


def on_open(ws):
    print("Opened connection")
    ws.send(
        '{"jsonrpc": "2.0", "method": "subscribe", "params": ["tm.event=\'NewBlock\'"], "id": 1}'
    )
    print("Sent subscribe request")


# from websocket import create_connection
if __name__ == "__main__":
    websocket.enableTrace(False)  # toggle to show or hide output
    ws = websocket.WebSocketApp(
        f"{RPC_URL}",
        on_open=on_open,
        on_message=on_message,
        on_error=on_error,
        on_close=on_close,
    )

    ws.run_forever(
        dispatcher=rel, reconnect=5
    )  # Set dispatcher to automatic reconnection, 5 second reconnect delay if connection closed unexpectedly
    rel.signal(2, rel.abort)  # Keyboard Interrupt
    rel.dispatch()
