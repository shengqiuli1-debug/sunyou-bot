#!/usr/bin/env node
/* eslint-disable no-console */

const BASE = process.env.SUNYOU_BASE || "http://127.0.0.1";
const ROLE = process.env.SUNYOU_E2E_ROLE || "narrator";
const FIRE = process.env.SUNYOU_E2E_FIRE || "high";
const MUTE_SECONDS = Number(process.env.SUNYOU_E2E_MUTE_SECONDS || "6");

function nowTag() {
  return new Date().toISOString().slice(11, 19).replaceAll(":", "");
}

async function api(path, { method = "GET", token = "", body } = {}) {
  const headers = { "Content-Type": "application/json" };
  if (token) headers["X-User-Token"] = token;

  const res = await fetch(`${BASE}/api/v1${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  const text = await res.text();
  let data = {};
  try {
    data = text ? JSON.parse(text) : {};
  } catch {
    data = { raw: text };
  }
  if (!res.ok) {
    throw new Error(`${method} ${path} -> ${res.status}: ${JSON.stringify(data)}`);
  }
  return data;
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function connectWS(roomId, token, label) {
  return new Promise((resolve, reject) => {
    if (typeof WebSocket === "undefined") {
      reject(new Error("当前 Node 环境没有 WebSocket 全局对象，请升级到 Node 20+"));
      return;
    }
    const wsUrl = `${BASE.replace("http://", "ws://").replace("https://", "wss://")}/ws/rooms/${roomId}?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(wsUrl);
    const events = [];
    const timer = setTimeout(() => {
      try {
        ws.close();
      } catch {}
      reject(new Error(`${label} websocket connect timeout`));
    }, 6000);

    ws.onopen = () => {
      clearTimeout(timer);
      resolve({ ws, label, events });
    };
    ws.onerror = () => reject(new Error(`${label} websocket error`));
    ws.onmessage = (ev) => {
      try {
        events.push(JSON.parse(ev.data));
      } catch {
        events.push({ type: "raw", payload: String(ev.data) });
      }
    };
  });
}

function findBotMessages(events) {
  return events.filter((e) => e.type === "chat" && e.payload && e.payload.senderType === "bot");
}

async function main() {
  const tag = nowTag();
  console.log(`[E2E] BASE=${BASE}`);
  console.log("[E2E] Step 1/9 health check ...");
  const health = await fetch(`${BASE}/healthz`);
  if (!health.ok) {
    throw new Error(`healthz failed: ${health.status}`);
  }

  console.log("[E2E] Step 2/9 create two guests ...");
  const u1 = await api("/users/guest", { method: "POST", body: { nickname: `验收A${tag}` } });
  const u2 = await api("/users/guest", { method: "POST", body: { nickname: `验收B${tag}` } });

  console.log("[E2E] Step 3/9 create room by user A ...");
  const create = await api("/rooms", {
    method: "POST",
    token: u1.token,
    body: {
      durationMinutes: 5,
      botRole: "judge",
      fireLevel: "medium",
      generateReport: true,
    },
  });
  const roomId = create.room.id;

  console.log("[E2E] Step 4/9 user B join as target + identity switches ...");
  const join2 = await api(`/rooms/${roomId}/join`, {
    method: "POST",
    token: u2.token,
    body: { identity: "target", confirmTarget: true },
  });
  const immune = await api(`/rooms/${roomId}/identity`, {
    method: "POST",
    token: u2.token,
    body: { identity: "immune", confirmTarget: false },
  });
  const targetAgain = await api(`/rooms/${roomId}/identity`, {
    method: "POST",
    token: u2.token,
    body: { identity: "target", confirmTarget: true },
  });

  console.log("[E2E] Step 5/9 host controls (role/fire/mute) ...");
  await api(`/rooms/${roomId}/control`, {
    method: "POST",
    token: u1.token,
    body: { action: "switch_role", value: ROLE },
  });
  await api(`/rooms/${roomId}/control`, {
    method: "POST",
    token: u1.token,
    body: { action: "switch_fire", value: FIRE },
  });
  await api(`/rooms/${roomId}/control`, {
    method: "POST",
    token: u1.token,
    body: { action: "mute_bot", value: String(MUTE_SECONDS) },
  });

  console.log("[E2E] Step 6/9 open websocket for A/B ...");
  const c1 = await connectWS(roomId, u1.token, "A");
  const c2 = await connectWS(roomId, u2.token, "B");

  console.log("[E2E] Step 7/9 send messages and wait bot ...");
  c2.ws.send(JSON.stringify({ type: "chat", content: "都行？稳了。" }));
  await sleep(1200);
  await sleep(MUTE_SECONDS * 1000 + 200);
  c2.ws.send(JSON.stringify({ type: "chat", content: "都行，立个flag，这把不可能翻车？" }));
  await sleep(2200);
  c2.ws.send(JSON.stringify({ type: "chat", content: "我懂了，你们继续。" }));
  await sleep(2200);

  const botMessages = [...findBotMessages(c1.events), ...findBotMessages(c2.events)];

  console.log("[E2E] Step 8/9 end room + fetch report ...");
  const endRes = await api(`/rooms/${roomId}/end`, { method: "POST", token: u1.token });
  const reportRes = await api(`/rooms/${roomId}/report`, { method: "GET", token: u1.token });

  c1.ws.close();
  c2.ws.close();

  console.log("[E2E] Step 9/9 summary");
  const summary = {
    roomId,
    shareCode: create.room.shareCode,
    userA: u1.user.nickname,
    userB: u2.user.nickname,
    joinIdentity: join2.member.identity,
    switchedToImmune: immune.member.identity,
    switchedBackTarget: targetAgain.member.identity,
    wsEventsA: c1.events.length,
    wsEventsB: c2.events.length,
    botMessageCount: botMessages.length,
    sampleBotMessage: botMessages[0]?.payload?.content || null,
    reportQuote: endRes.report?.botQuote || reportRes.report?.botQuote || null,
    report: reportRes.report,
  };
  console.log(JSON.stringify(summary, null, 2));

  if (botMessages.length === 0) {
    throw new Error("botMessageCount = 0，建议重跑一次（触发概率逻辑存在随机性）");
  }
}

main().catch((err) => {
  console.error("[E2E_FAIL]", err.message);
  process.exit(1);
});

