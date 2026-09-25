import http from "k6/http";
import { check, sleep } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

const TEST_EMAIL = __ENV.TEST_EMAIL || "bench@example.com";
const TEST_PASSWORD = __ENV.TEST_PASSWORD || "bench_password_123";

export const options = {
  stages: [
    { duration: "10s", target: 20 },
    { duration: "30s", target: 50 },
    { duration: "10s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<500"],
    http_req_failed: ["rate<0.01"],
  },
};

export default function () {
  const res = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: TEST_EMAIL, password: TEST_PASSWORD }),
    { headers: { "Content-Type": "application/json" } }
  );
  check(res, {
    "status 200": (r) => r.status === 200,
    "has access_token": (r) => JSON.parse(r.body).access_token !== undefined,
  });
  sleep(1);
}
