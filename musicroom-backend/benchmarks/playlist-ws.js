import http from "k6/http";
import { check, sleep } from "k6";
import { Counter } from "k6/metrics";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TOKEN = __ENV.TOKEN || "";
const PLAYLIST_ID = __ENV.PLAYLIST_ID || "";
const TRACK_ID = __ENV.TRACK_ID || "";

const conflicts = new Counter("playlist_conflicts");
const retries = new Counter("playlist_retries");
const unresolved = new Counter("playlist_unresolved");

const MAX_RETRIES = 5;
const BACKOFF_BASE_MS = 20;

export const options = {
  stages: [
    { duration: "5s", target: 5 },
    { duration: "20s", target: 20 },
    { duration: "5s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<1000"],
    playlist_conflicts: [],
    playlist_unresolved: ["count==0"],
  },
};

function getPlaylist() {
  const res = http.get(`${BASE_URL}/playlists/${PLAYLIST_ID}`, {
    headers: { Authorization: `Bearer ${TOKEN}` },
  });
  check(res, { "get 200": (r) => r.status === 200 });
  return JSON.parse(res.body);
}

function moveTrack(version) {
  return http.patch(
    `${BASE_URL}/playlists/${PLAYLIST_ID}/tracks/${TRACK_ID}`,
    JSON.stringify({ after_track_id: null, version }),
    {
      headers: {
        Authorization: `Bearer ${TOKEN}`,
        "Content-Type": "application/json",
      },
    }
  );
}

export default function () {
  let playlist = getPlaylist();
  let version = playlist.version;

  let res = moveTrack(version);

  if (res.status === 409) {
    conflicts.add(1);
    for (let attempt = 0; attempt < MAX_RETRIES && res.status === 409; attempt++) {
      const span = BACKOFF_BASE_MS * 2 ** attempt;
      sleep((span + Math.random() * span * 1.5) / 1000);
      retries.add(1);
      playlist = getPlaylist();
      version = playlist.version;
      res = moveTrack(version);
    }
    if (res.status === 409) unresolved.add(1);
  }

  check(res, { "move succeeded": (r) => r.status === 200 });
  sleep(0.5);
}
