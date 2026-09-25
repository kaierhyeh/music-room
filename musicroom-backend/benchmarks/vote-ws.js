import ws from "k6/ws";
import http from "k6/http";
import { check, sleep } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const WS_URL = BASE_URL.replace(/^http/, "ws");
const TOKEN = __ENV.TOKEN || "";
const EVENT_ID = __ENV.EVENT_ID || "";
const TRACK_ID = __ENV.TRACK_ID || "";

export const options = {
  stages: [
    { duration: "5s", target: 10 },
    { duration: "30s", target: 100 },
    { duration: "5s", target: 0 },
  ],
  thresholds: {
    ws_connecting: ["p(95)<500"],
    http_req_duration: ["p(95)<300"],
  },
};

export default function () {
  const ticketRes = http.post(`${BASE_URL}/ws-tickets`, null, {
    headers: { Authorization: `Bearer ${TOKEN}` },
  });
  check(ticketRes, { "ticket issued": (r) => r.status === 200 });
  const { ticket } = JSON.parse(ticketRes.body);

  let received = false;
  const wsRes = ws.connect(
    `${WS_URL}/ws/events/${EVENT_ID}?ticket=${ticket}`,
    {},
    function (socket) {
      socket.on("message", (data) => {
        const msg = JSON.parse(data);
        if (msg.type === "queue:updated") {
          received = true;
          socket.close();
        }
      });

      socket.setTimeout(() => {
        http.post(
          `${BASE_URL}/events/${EVENT_ID}/tracks/${TRACK_ID}/vote`,
          null,
          { headers: { Authorization: `Bearer ${TOKEN}` } }
        );
      }, 100);

      socket.setTimeout(() => socket.close(), 3000);
    }
  );

  check(wsRes, { "ws status 101": (r) => r && r.status === 101 });
  sleep(1);
}
