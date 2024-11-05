function showPopup(popupId) {
    var popup = document.getElementById(popupId);
    popup.style.display = "block";
}

function closePopup(popupId) {
    var popup = document.getElementById(popupId);
    popup.style.display = "none";
}

const popupIds = ["loginPopup", "signupPopup"];

popupIds.forEach(popupId => {
  const popupContainer = document.getElementById(popupId);

  if (popupContainer) {
    popupContainer.addEventListener("click", event => {
      if (event.target === popupContainer) {
        popupContainer.style.display = "none";
      }
    });
  }
});



function filterPosts(filterType, filterValue) {
  let url = "/?";
  if (filterType === "category") {
      url += "category=" + encodeURIComponent(filterValue);
  } else {
      url += "filter=" + filterValue;
  }
  window.location.href = url;
}

