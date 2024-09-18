var params = new URLSearchParams(location.search);
var gameId = params.get('game');

var conn = new WebSocket(`ws://${location.host}/ws/${gameId}`);

conn.onclose = function() {
    alert('Connection lost');
};

conn.onmessage = function(event) {
    // TODO
    console.log(JSON.parse(event.data));
};
