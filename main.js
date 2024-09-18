var conn = new WebSocket(`ws://${location.host}/ws/`);

conn.onclose = function() {
    alert('Connection lost');
};

conn.onmessage = function(event) {
    // TODO
    console.log(event.data);
};
