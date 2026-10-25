function editPracticeMatch(button) {
  const data = button.dataset;
  document.getElementById("editMatchId").value = data.id;
  document.getElementById("editMatchRevision").value = data.revision;
  document.getElementById("editMatchLabel").textContent = data.label;
  document.getElementById("editMatchTime").value = data.time;
  ["red1", "red2", "red3", "blue1", "blue2", "blue3"].forEach(function (station) {
    const input = document.getElementById("edit" + station.charAt(0).toUpperCase() + station.slice(1));
    input.value = Number(data[station]) || "";
  });
}
