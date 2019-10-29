var exposureTables = document.querySelectorAll(".exposureBlock");

var showExposureTable = function(elements, index){
    elements.forEach(function(table) {
        table.classList.toggle("hidden", true);
    });
    elements[index].classList.toggle("hidden", false);
}

var i = 1;
window.setInterval(function() {
    if( i == exposureTables.length ) {
        location.reload()
    }
    var active = i
    showExposureTable(exposureTables, active);
    i++;
}, 1000);