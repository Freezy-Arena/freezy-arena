// Copyright 2026 Team 254. All Rights Reserved.

// Client-side methods for the unpicked teams display.

$(function () {
  // Read the configuration for this display from the URL query string.
  const urlParams = new URLSearchParams(window.location.search);

  if (urlParams.get("inverted") === "true") {
    $("#unpickedTeamsContainer").css("transform", "rotate(180deg)");
    $("#gameLogoContainer").css("transform", "rotate(180deg)");
  }

  const $unpickedTeams = $("#unpickedTeams");
  const $container = $("#unpickedTeamsContainer");
  const $logoContainer = $("#gameLogoContainer");
  const unpickedTeamsTemplate = Handlebars.compile($("#unpickedTeamsTemplate").html());

  let currentRankedTeams = [];
  let isAllianceSelection = false;

  const updateDisplay = function() {
    if (isAllianceSelection && currentRankedTeams.length > 0) {
      $container.css("display", "flex");
      $logoContainer.hide();
      
      const unpickedTeams = currentRankedTeams.filter(team => !team.Picked);
      $unpickedTeams.html(unpickedTeamsTemplate(unpickedTeams));
    } else {
      $container.hide();
      $logoContainer.css("display", "flex");
    }
  };

  new CheesyWebsocket("/displays/unpicked/websocket", {
    allianceSelection: function (event) {
      currentRankedTeams = event.data.RankedTeams || [];
      updateDisplay();
    },
    audienceDisplayMode: function (event) {
      isAllianceSelection = event.data === "allianceSelection";
      updateDisplay();
    }
  });
});


