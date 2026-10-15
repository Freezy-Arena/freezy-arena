// Live standings and playoff matchups for the audience output.
let audienceInfoMode = null;
let audienceInfoRows = [];
let audienceInfoPage = 0;
let audienceInfoHighestMatch = "";
let audienceInfoDisplayFields = ["rp"];
let audienceInfoNextMatches = {};
let audienceInfoHasLoaded = false;
let audienceInfoRefreshTimer;
let audienceInfoPageTimer;
let audienceInfoGeneration = 0;
let audienceInfoTickerStartMs = 0;
let audienceInfoTickerDurationMs = 0;

// The compact bracket still rotates pages; both rankings modes scroll continuously.
const audienceInfoPageSizes = {bracketBar: 3};
const audienceInfoTickerPixelsPerSecond = 72;
const audienceInfoFullPixelsPerSecond = 30;
let audienceInfoFullStartMs = 0;
let audienceInfoFullDurationMs = 0;

const audienceRankingsLabels = {
  rp: "RP", nextMatch: "Next", record: "W-L-T", played: "Played",
  matchPoints: "Match Pts", autoFuel: "Auto Fuel", tower: "Tower"
};

function audienceRankingsValue(key, rank) {
  switch (key) {
    case "rp": return rank.RankingPoints ?? "—";
    case "nextMatch": return audienceInfoNextMatches[rank.TeamId] || "—";
    case "record": return (rank.Wins || 0) + "-" + (rank.Losses || 0) + "-" + (rank.Ties || 0);
    case "played": return rank.Played ?? 0;
    case "matchPoints": return rank.MatchPoints ?? 0;
    case "autoFuel": return rank.AutoFuelPoints ?? 0;
    case "tower": return rank.TowerPoints ?? 0;
    default: return "";
  }
}

function audienceInfoSpan(text, className) {
  return $("<span>").addClass(className || "").text(text == null ? "" : text);
}

// Keep the animation at the same relative position when rankings update mid-scroll.
function audienceInfoTickerProgress() {
  if (!audienceInfoTickerDurationMs) return 0;
  const elapsed = Date.now() - audienceInfoTickerStartMs;
  return ((elapsed % audienceInfoTickerDurationMs) + audienceInfoTickerDurationMs) %
    audienceInfoTickerDurationMs / audienceInfoTickerDurationMs;
}

function renderAudienceRankingsTicker(content, rows) {
  const previousProgress = audienceInfoTickerProgress();
  const viewport = $("<div>").addClass("audience-info-ticker-viewport");
  const track = $("<div>").addClass("audience-info-ticker-track");
  const group = $("<div>").addClass("audience-info-ticker-group");
  rows.forEach(function (row) {
    const card = $("<div>").addClass("audience-info-ticker-card");
    card.append(audienceInfoSpan("#" + row.Rank + ":", "audience-info-ticker-rank"));
    const team = $("<div>").addClass("audience-info-ticker-team");
    team.append(audienceInfoSpan(row.TeamId, "audience-info-ticker-team-id"));
    team.append(audienceInfoSpan(row.Nickname, "audience-info-ticker-nickname"));
    card.append(team);
    const metrics = $("<div>").addClass("audience-info-ticker-metrics");
    audienceInfoDisplayFields.forEach(function (key) {
      metrics.append(audienceInfoSpan(
        audienceRankingsLabels[key] + " " + audienceRankingsValue(key, row),
        "audience-info-ticker-points"
      ));
    });
    card.append(metrics);
    group.append(card);
  });
  track.append(group);
  content.append(viewport.append(track));

  // Measure the actual rendered width; don't scroll if every team already fits.
  const distance = group[0].getBoundingClientRect().width;
  if (distance > viewport[0].clientWidth + 1) {
    // Duplicate the exact sequence, including trailing spacing, for a seamless wrap.
    track.append(group.clone());
    const durationMs = distance / audienceInfoTickerPixelsPerSecond * 1000;
    const elapsedMs = previousProgress * durationMs;
    track.css({
      "--audience-info-scroll-duration": durationMs + "ms",
      "animation-delay": -elapsedMs + "ms"
    }).addClass("is-scrolling");
    audienceInfoTickerDurationMs = durationMs;
    audienceInfoTickerStartMs = Date.now() - elapsedMs;
  } else {
    audienceInfoTickerDurationMs = 0;
    audienceInfoTickerStartMs = 0;
  }
}

// Preserve relative scroll progress across live score or option changes.
function audienceInfoFullProgress() {
  if (!audienceInfoFullDurationMs) return 0;
  const elapsed = Date.now() - audienceInfoFullStartMs;
  return ((elapsed % audienceInfoFullDurationMs) + audienceInfoFullDurationMs) %
    audienceInfoFullDurationMs / audienceInfoFullDurationMs;
}

// Fixed table headers remain readable while every ranking moves vertically.
function renderAudienceRankingsFull(content, rows) {
  const previousProgress = audienceInfoFullProgress();
  const headings = ["Rank", "Team", "Team Name"].concat(
    audienceInfoDisplayFields.map(function (key) { return audienceRankingsLabels[key]; })
  );

  // Keep header and rows aligned as operators add or remove ranking columns.
  const widths = [8, 13, 79 - 14 * audienceInfoDisplayFields.length].concat(
    audienceInfoDisplayFields.map(function () { return 14; })
  );
  function newTable(className) {
    const table = $("<table>").addClass("audience-info-table " + className);
    const colgroup = $("<colgroup>");
    widths.forEach(function (width) {
      colgroup.append($("<col>").css("width", width + "%"));
    });
    return table.append(colgroup);
  }

  const header = newTable("audience-info-full-header");
  const headingRow = $("<tr>");
  headings.forEach(function (heading) { headingRow.append($("<th>").text(heading)); });
  header.append($("<thead>").append(headingRow));

  const table = newTable("audience-info-full-data");
  const body = $("<tbody>");
  rows.forEach(function (rank) {
    const tr = $("<tr>");
    [rank.Rank, rank.TeamId, rank.Nickname].concat(
      audienceInfoDisplayFields.map(function (key) { return audienceRankingsValue(key, rank); })
    ).forEach(function (value) {
      tr.append($("<td>").text(value == null ? "" : value));
    });
    body.append(tr);
  });
  table.append(body);

  const viewport = $("<div>").addClass("audience-info-full-viewport");
  const track = $("<div>").addClass("audience-info-full-track");
  const group = $("<div>").addClass("audience-info-full-group").append(table);
  track.append(group);
  content.append(header, viewport.append(track));

  const distance = group[0].getBoundingClientRect().height;
  const available = viewport[0].clientHeight;
  if (available > 0 && distance > available + 1) {
    // One repeated copy makes the vertical loop seamless and gap-free.
    track.append(group.clone());
    const durationMs = distance / audienceInfoFullPixelsPerSecond * 1000;
    const elapsedMs = previousProgress * durationMs;
    track.css({
      "--audience-info-full-duration": durationMs + "ms",
      "animation-delay": -elapsedMs + "ms"
    }).addClass("is-scrolling");
    audienceInfoFullDurationMs = durationMs;
    audienceInfoFullStartMs = Date.now() - elapsedMs;
  } else {
    // Keep short leaderboards stationary if all of their rows fit the screen.
    audienceInfoFullDurationMs = 0;
    audienceInfoFullStartMs = 0;
  }
  $("#audienceInformationFooter").text(
    rows.length + " TEAMS" + (audienceInfoHighestMatch ?
      " · Standings as of " + audienceInfoHighestMatch : "")
  );
}

function renderAudienceInformation() {
  if (!audienceInfoMode) return;
  const mode = audienceInfoMode, rows = audienceInfoRows;
  const content = $("#audienceInformationContent").empty();
  if (!rows.length) {
    content.append($("<div>").addClass("audience-info-empty").text(
      mode === "bracketBar" ? "No playoff matches scheduled." : "No qualification standings available yet."
    ));
    $("#audienceInformationFooter").empty();
    return;
  }
  if (mode === "rankingsBar") {
    renderAudienceRankingsTicker(content, rows);
    $("#audienceInformationFooter").text(
      rows.length + " TEAMS" + (audienceInfoHighestMatch ?
        " · Standings as of " + audienceInfoHighestMatch : "")
    );
    return;
  }

  if (mode === "rankingsFull") {
    renderAudienceRankingsFull(content, rows);
    return;
  }

  // Allow each matchup enough room for alliance names and scores.
  // Full-width outputs show three; smaller displays rotate one or two at a time.
  const size = mode === "bracketBar"
    ? (window.innerWidth < 900 ? 1 : window.innerWidth < 1400 ? 2 : audienceInfoPageSizes.bracketBar)
    : audienceInfoPageSizes[mode];
  const pageCount = Math.ceil(rows.length / size);
  audienceInfoPage %= pageCount;
  const first = audienceInfoPage * size;
  const visible = rows.slice(first, first + size);
  if (mode === "bracketBar") {
    const cards = $("<div>").addClass("audience-info-cards");
    visible.forEach(function (row) {
      const card = $("<div>").addClass("audience-info-card");
      {
        card.addClass("audience-info-playoff");
        card.append(audienceInfoSpan(row.LongName || row.ShortName, "audience-info-match-name"));
        const sides = $("<div>").addClass("audience-info-sides");
        [["red", row.PlayoffRedAlliance, [row.Red1,row.Red2,row.Red3], "RedSummary"],
          ["blue", row.PlayoffBlueAlliance, [row.Blue1,row.Blue2,row.Blue3], "BlueSummary"]
        ].forEach(function (side, index) {
          const label = side[1] ? "Alliance " + side[1] :
            (side[2].filter(Boolean).join(" / ") || "TBD");
          const score = row.Result && row.Result[side[3]] && row.Result[side[3]].Score;
          const box = $("<div>").addClass("audience-info-side " + side[0]);
          box.append(audienceInfoSpan(label));
          if (score != null) box.append(audienceInfoSpan(score, "audience-info-score"));
          if (index) sides.append(audienceInfoSpan("VS", "audience-info-vs"));
          sides.append(box);
        });
        card.append(sides);
      }
      cards.append(card);
    });
    content.append(cards);
  }
  const suffix = audienceInfoHighestMatch && mode !== "bracketBar" ?
    " · Standings as of " + audienceInfoHighestMatch : "";
  $("#audienceInformationFooter").text(
    "Showing " + (first + 1) + "–" + (first + visible.length) + " of " + rows.length + suffix
  );
}

function refreshAudienceInformation() {
  if (!audienceInfoMode) return;
  const mode = audienceInfoMode, generation = audienceInfoGeneration;
  $.getJSON(mode === "bracketBar" ? "/api/matches/playoff" : "/api/rankings")
    .done(function (data) {
      if (mode !== audienceInfoMode || generation !== audienceInfoGeneration) return;
      const next = mode === "bracketBar" ? data : data.Rankings;
      const newRows = Array.isArray(next) ? next : [];
      const highest = mode === "bracketBar" ? "" : data.HighestPlayedMatch;
      const fields = mode === "bracketBar" ? [] :
        (Array.isArray(data.DisplayFields) ? data.DisplayFields : ["rp"]);
      const nextMatches = mode === "bracketBar" ? {} : (data.NextMatches || {});
      if (!audienceInfoHasLoaded || JSON.stringify(newRows) !== JSON.stringify(audienceInfoRows) ||
          highest !== audienceInfoHighestMatch ||
          JSON.stringify(fields) !== JSON.stringify(audienceInfoDisplayFields) ||
          JSON.stringify(nextMatches) !== JSON.stringify(audienceInfoNextMatches)) {
        audienceInfoHasLoaded = true;
        audienceInfoRows = newRows;
        audienceInfoHighestMatch = highest;
        audienceInfoDisplayFields = fields;
        audienceInfoNextMatches = nextMatches;
        renderAudienceInformation();
      }
    }).fail(function () {
      if (mode === audienceInfoMode && generation === audienceInfoGeneration &&
          !audienceInfoRows.length) {
        $("#audienceInformationContent").text("Unable to load live data. Retrying...");
      }
    });
}

function showAudienceInformation(mode, callback) {
  clearInterval(audienceInfoRefreshTimer);
  clearInterval(audienceInfoPageTimer);
  audienceInfoGeneration++;
  audienceInfoMode = mode;
  audienceInfoRows = [];
  audienceInfoDisplayFields = ["rp"];
  audienceInfoNextMatches = {};
  audienceInfoHasLoaded = false;
  audienceInfoHighestMatch = "";
  audienceInfoPage = 0;
  audienceInfoTickerStartMs = 0;
  audienceInfoTickerDurationMs = 0;
  audienceInfoFullStartMs = 0;
  audienceInfoFullDurationMs = 0;
  $("#audienceInformation")
    .attr("class", mode === "rankingsFull" ? "full" : "bar")
    .attr("data-mode", mode);
  $("#audienceInformationTitle").text(mode === "bracketBar" ? "PLAYOFF BRACKET" : "TEAM STANDINGS");
  $("#audienceInformationContent").text("Loading...");
  $("#audienceInformationFooter").empty();
  // Display the correct layout before measuring ticker widths. jQuery's
  // fadeIn() defaults to block and can collapse the heading into the cards.
  $("#audienceInformation")
    .css({display: mode === "rankingsFull" ? "flex" : "grid", opacity: 0})
    .animate({opacity: 1}, 250, callback);
  refreshAudienceInformation();
  audienceInfoRefreshTimer = setInterval(refreshAudienceInformation, 15000);
  if (mode === "bracketBar") {
    audienceInfoPageTimer = setInterval(function () {
      if (!audienceInfoRows.length) return;
      audienceInfoPage++;
      renderAudienceInformation();
    }, 8500);
  }
}

function hideAudienceInformation(callback) {
  clearInterval(audienceInfoRefreshTimer);
  clearInterval(audienceInfoPageTimer);
  audienceInfoGeneration++;
  audienceInfoMode = null;
  audienceInfoTickerDurationMs = 0;
  audienceInfoTickerStartMs = 0;
  audienceInfoFullDurationMs = 0;
  audienceInfoFullStartMs = 0;
  $("#audienceInformation").fadeOut(250, callback);
}

// Recalculate scroll distance and layout after output resolution changes.
$(window).on("resize.audienceInformation", function () {
  if (audienceInfoMode && audienceInfoRows.length) {
    renderAudienceInformation();
  }
});

