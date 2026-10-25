// Keep the queue operator's view current without discarding unsaved form edits.
// The server returns only an authenticated revision token, not the kiosk data.
(function () {
  "use strict";
  var currentRevision = window.fillerQueueRevision;
  if (!currentRevision) return;

  var dirty = false;
  var checking = false;
  var status = document.getElementById("fillerLiveStatus");

  document.addEventListener("input", function (event) {
    if (event.target.closest("form")) dirty = true;
  });
  document.addEventListener("change", function (event) {
    if (event.target.closest("form")) dirty = true;
  });

  function editing() {
    var active = document.activeElement;
    return dirty || document.querySelector(".modal.show") !== null ||
      (active && active.closest("form") && active !== document.body);
  }

  async function checkForChanges() {
    if (checking || document.visibilityState === "hidden") return;
    checking = true;
    try {
      var response = await fetch("/panel/freezy/filler/revision", {
        credentials: "same-origin",
        cache: "no-store",
        headers: {"Accept": "application/json"}
      });
      if (!response.ok || !response.headers.get("content-type").includes("application/json")) {
        throw new Error("Revision check unavailable");
      }
      var data = await response.json();
      if (data.revision !== currentRevision) {
        if (!editing()) {
          window.location.reload();
          return;
        }
        if (status) {
          status.textContent = "Schedule updated — finish editing, then refresh to see changes.";
          status.classList.remove("text-success");
          status.classList.add("text-warning");
        }
      } else if (status && !dirty) {
        status.textContent = "Live updates enabled — checking every 3 seconds";
      }
    } catch (error) {
      if (status && !dirty) {
        status.textContent = "Live updates paused — retrying automatically";
        status.classList.remove("text-success");
        status.classList.add("text-warning");
      }
    } finally {
      checking = false;
    }
  }

  window.setInterval(checkForChanges, 3000);
  window.addEventListener("focus", checkForChanges);
})();
