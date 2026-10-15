// Run with: node --test tests/js/audience_victory_video.test.cjs
const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

const file = fs.readFileSync(path.join(__dirname, "../../static/js/audience_display.js"), "utf8");
const start = file.indexOf("const playVictoryVideo = function (callback) {");
const end = file.indexOf("const getAvatarUrl = function", start);
assert.ok(start >= 0 && end > start, "Winner reveal function must exist");
const implementation = file.slice(start, end);

function createHarness(outcomes, winner = "red") {
  let now = 0, nextId = 0, attempts = 0, callbackCount = 0;
  const timers = new Map();
  const listeners = new Map();
  const overlay = {visible: false, show() {this.visible = true;}, hide() {this.visible = false;}};
  const video = {
    muted: false, src: "", currentTime: 0, error: null, paused: false,
    addEventListener(event, callback) { listeners.set(event, callback); },
    removeEventListener(event, callback) {
      if (listeners.get(event) === callback) listeners.delete(event);
    },
    emit(event) { listeners.get(event)?.(); },
    pause() {this.paused = true;},
    load() {},
    removeAttribute(attribute) {if (attribute === "src") this.src = "";},
    play() {
      const outcome = outcomes[Math.min(attempts++, outcomes.length - 1)];
      if (outcome === "blocked") {
        const error = new Error("Autoplay blocked");
        error.name = "NotAllowedError";
        return Promise.reject(error);
      }
      if (outcome === "unsupported") return Promise.reject(new Error("Unsupported codec"));
      if (outcome === "pending") return new Promise(() => {});
      return Promise.resolve();
    },
  };
  const context = vm.createContext({
    winner,
    console: {warn() {}, error() {}},
    $: (selector) => selector === "#videoOverlay" ? overlay : [video],
    setTimeout(callback, delay) {
      const id = ++nextId;
      timers.set(id, {at: now + delay, callback});
      return id;
    },
    clearTimeout(id) {timers.delete(id);},
  });
  vm.runInContext(implementation + "\nthis.playVictoryVideo = playVictoryVideo;", context);

  function advance(milliseconds) {
    const target = now + milliseconds;
    while (true) {
      const due = [...timers.entries()]
        .filter(([, value]) => value.at <= target)
        .sort((a, b) => a[1].at - b[1].at)[0];
      if (!due) break;
      now = due[1].at;
      timers.delete(due[0]);
      due[1].callback();
    }
    now = target;
  }

  return {
    video, overlay, advance,
    start() {context.playVictoryVideo(() => callbackCount++);},
    get attempts() {return attempts;},
    get callbackCount() {return callbackCount;},
    get remainingTimers() {return timers.size;},
  };
}

test("winner reveal plays with audio and finishes once after video ends", async () => {
  const h = createHarness(["ok"], "blue");
  h.start();
  assert.equal(h.video.src, "/static/video/BlueWins.mp4");
  assert.equal(h.overlay.visible, true);
  assert.equal(h.video.muted, false);
  h.video.emit("playing");
  h.video.currentTime = 1;
  h.video.emit("timeupdate");
  h.video.emit("ended");
  h.advance(499);
  assert.equal(h.callbackCount, 0);
  h.advance(1);
  assert.equal(h.callbackCount, 1);
  assert.equal(h.overlay.visible, false);
  assert.equal(h.video.src, "");
  h.advance(30000);
  assert.equal(h.callbackCount, 1);
});

test("autoplay policy falls back to muted playback and completes", async () => {
  const h = createHarness(["blocked", "ok"]);
  h.start();
  await Promise.resolve();
  await Promise.resolve();
  assert.equal(h.attempts, 2);
  assert.equal(h.video.muted, true);
  h.video.emit("playing");
  h.video.emit("ended");
  h.advance(500);
  assert.equal(h.callbackCount, 1);
  assert.equal(h.overlay.visible, false);
});

test("unsupported media skips the black screen and advances the final score", async () => {
  const h = createHarness(["unsupported"]);
  h.start();
  await Promise.resolve();
  assert.equal(h.callbackCount, 1);
  assert.equal(h.overlay.visible, false);
  assert.equal(h.remainingTimers, 0);
});

test("pending autoplay is stopped after the start timeout", () => {
  const h = createHarness(["pending"]);
  h.start();
  h.advance(12000);
  assert.equal(h.callbackCount, 1);
  assert.equal(h.overlay.visible, false);
  assert.equal(h.remainingTimers, 0);
});

test("stalled video advances without hanging behind a black screen", () => {
  const h = createHarness(["ok"], "tie");
  h.start();
  assert.equal(h.video.src, "/static/video/Tie.mp4");
  h.video.emit("playing");
  h.video.currentTime = 1;
  h.video.emit("timeupdate");
  h.advance(15000);
  assert.equal(h.callbackCount, 1);
  assert.equal(h.overlay.visible, false);
  assert.equal(h.remainingTimers, 0);
});
