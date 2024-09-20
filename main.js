var $pre = document.querySelector('pre');

var params = new URLSearchParams(location.search);
var gameId = params.get('game');

var conn = new WebSocket(`ws://${location.host}/ws/${gameId}`);

var game = {
    id: -1,
    rects: [],
    objects: {},

    getChar(x, y) {
        if (Object.values(this.objects).some(obj => x === obj.pos.x && y === obj.pos.y)) {
            return ['@', 1];
        }
        if (this.rects.some(rect => x >= rect.x1 && x <= rect.x2 && y >= rect.y1 && y <= rect.y2)) {
            return ['.', -1];
        }
        if (this.rects.some(rect => x >= rect.x1 - 1 && x <= rect.x2 + 1 && y >= rect.y1 - 1 && y <= rect.y2 + 1)) {
            return ['#', -1];
        }
        return [' ', -1];
    },
};

var getSize = function() {
    // minimum is 10x10
    // maximum is 100x30
    // consider aspect ratio
    // find font size and rows/columns for best match
    // probably have to experiment
    return [10, 20, 100];
};

var render = function() {
    var [fontSize, rows, cols] = getSize();

    var xOffset = -(cols >> 1);
    var yOffset = -(rows >> 1);
    if (game.objects[game.id]) {
        xOffset += game.objects[game.id].pos.x;
        yOffset += game.objects[game.id].pos.y;
    }

    $pre.style.fontSize = fontSize;
    $pre.innerHTML = '';
    var spanColor = -1;
    var span = '';

    var commitSpan = () => {
        if (spanColor === -1) {
            $pre.append(span);
        } else {
            var $span = document.createElement('span');
            $span.innerText = span;
            $span.className = `color-${spanColor}`;
            $pre.append($span);
        }
        span = '';
        spanColor = -1;
    };

    for (let y = 0; y < rows; y++) {
        for (let x = 0; x < cols; x++) {
            const [c, color] = game.getChar(xOffset + x, yOffset + y);
            if (color === spanColor) {
                span += c;
            } else {
                commitSpan();
                span = c;
                spanColor = color;
            }
        }

        commitSpan();
        $pre.append('\n');
    }
};

conn.onclose = function() {
    alert('Connection lost');
};

conn.onmessage = function(event) {
    var messages = JSON.parse(event.data);
    for (const msg of messages) {
        console.log(msg);
        if (msg.action === 'setId') {
            game.id = msg.id;
        } else if (msg.action === 'setLevel') {
            game.rects = msg.rects;
            game.horizontal = msg.horizontal;
            game.vertical = msg.vertical;
        } else if (msg.action === 'create') {
            game.objects[msg.id] = {
                type: msg.type,
                pos: msg.pos,
            };
        }
    }
    render();
};
