var $pre = document.querySelector('pre');
var $dpad = document.querySelector('#dpad');
var radius = 5;

var params = new URLSearchParams(location.search);
var gameId = params.get('game');

var socketProtocol = location.protocol.replace('http', 'ws');
var socket = new WebSocket(`${socketProtocol}//${location.host}/ws/${gameId}`);
var pointer = null;

var send = function(data) {
    socket.send(JSON.stringify(data));
};

var COLORS = {
    'player': 4,
    'monster': 1,
};

var inRect = function(pos, rect, withWalls) {
    if (withWalls) {
        return pos.x >= rect.x1 - 1 && pos.x <= rect.x2 + 1
            && pos.y >= rect.y1 - 1 && pos.y <= rect.y2 + 1;
    } else {
        return pos.x >= rect.x1 && pos.x <= rect.x2
            && pos.y >= rect.y1 && pos.y <= rect.y2;
    }
};

var game = {
    id: -1,
    rects: [],
    seen: {},
    objects: {},
    health: 1,
    healthTotal: 1,

    getRect(pos, withWalls) {
        for (const rect of this.rects) {
            if (inRect(pos, rect, withWalls)) {
                return rect;
            }
        }
    },

    inView(a, b) {
        // check radius
        var dx = a.x - b.x;
        var dy = a.y - b.y;
        if (dx * dx + dy * dy >= radius * radius) {
            return false;
        }

        // perf: shortcut if in same rect
        for (const rect of this.rects) {
            if (
                inRect(a, rect, true) && inRect(b, rect, true)
                && (inRect(a, rect) || inRect(b, rect))
            ) {
                return true;
            }
        }

        // ray casting
        if (Math.abs(dx) > Math.abs(dy)) {
            const [c, d] = a.x > b.x ? [b, a] : [a, b];
            return [
                [c.y + 0.4, d.y + 0.4],
                [c.y + 0.4, d.y - 0.4],
                [c.y - 0.4, d.y + 0.4],
                [c.y - 0.4, d.y - 0.4],
            ].some(([y1, y2]) => {
                const f = (y2 - y1) / (d.x - c.x);
                for (let x = c.x + 1; x < d.x; x++) {
                    const y = Math.round((x - c.x) * f + y1);
                    if (!this.getRect({x, y})) {
                        return false;
                    }
                }
                return true;
            });
        } else {
            const [c, d] = a.y > b.y ? [b, a] : [a, b];
            return [
                [c.x + 0.4, d.x + 0.4],
                [c.x + 0.4, d.x - 0.4],
                [c.x - 0.4, d.x + 0.4],
                [c.x - 0.4, d.x - 0.4],
            ].some(([x1, x2]) => {
                const f = (x2 - x1) / (d.y - c.y);
                for (let y = c.y + 1; y < d.y; y++) {
                    const x = Math.round((y - c.y) * f + x1);
                    if (!this.getRect({x, y})) {
                        return false;
                    }
                }
                return true;
            });
        }
    },

    getChar(x, y) {
        if (!this.seen[[x, y]]) {
            return [' ', -1];
        }

        var inView = () => Object.values(this.objects).some(
            obj => obj.type === 'player' && this.inView(obj.pos, {x, y})
        );

        if (x === this.ladder.x && y === this.ladder.y) {
            return ['>', inView() ? -1 : 0];
        }
        var objs = Object.values(this.objects).filter(obj => x === obj.pos.x && y === obj.pos.y);
        for (const obj of objs) {
            if (obj.type === 'player' || inView()) {
                return [obj.rune, COLORS[obj.type]];
            }
        }
        if (this.getRect({x, y})) {
            return ['.', inView() ? -1 : 0];
        }
        if (this.getRect({x, y}, true)) {
            return ['#', inView() ? -1 : 0];
        }
        return [' ', -1];
    },

    updateSeen(pos) {
        for (let dy = -radius; dy <= radius; dy++) {
            const y = pos.y + dy;
            for (let dx = -radius; dx <= radius; dx++) {
                const x = pos.x + dx;
                if (!this.seen[[x, y]] && this.inView(pos, {x, y})) {
                    this.seen[[x, y]] = true;
                }
            }
        }
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

    var commitSpan = (text, color) => {
        if (color === -1) {
            $pre.append(text);
        } else {
            var $span = document.createElement('span');
            $span.innerText = text;
            $span.className = `color-${color}`;
            $pre.append($span);
        }
    };

    var health = Math.round(game.health / game.healthTotal * cols);
    commitSpan('='.repeat(health), 1);
    commitSpan('='.repeat(cols - health), 0);
    $pre.append('\n');

    for (let y = 1; y < rows; y++) {
        var span = '';
        var spanColor = -1;

        for (let x = 0; x < cols; x++) {
            const [c, color] = game.getChar(xOffset + x, yOffset + y);
            if (color === spanColor) {
                span += c;
            } else {
                commitSpan(span, spanColor);
                span = c;
                spanColor = color;
            }
        }

        commitSpan(span, spanColor);
        $pre.append('\n');
    }
};

socket.onclose = function() {
    alert('Connection lost');
};

socket.onmessage = function(event) {
    var messages = JSON.parse(event.data);
    for (const msg of messages) {
        if (msg.action === 'setId') {
            game.id = msg.id;
        } else if (msg.action === 'setLevel') {
            game.rects = msg.rects;
            game.ladder = msg.ladder;
            game.seen = {};
            for (const [id, obj] of Object.entries(game.objects)) {
                if (obj.type !== 'player') {
                    delete game.objects[id];
                }
            }
        } else if (msg.action === 'create') {
            game.objects[msg.id] = {
                type: msg.type,
                pos: msg.pos,
                rune: msg.rune,
            };
            if (msg.type === 'player') {
                game.updateSeen(msg.pos);
            }
        } else if (msg.action === 'setPosition') {
            game.objects[msg.id].pos = msg.pos;
            if (game.objects[msg.id].type === 'player') {
                game.updateSeen(msg.pos);
            }
        } else if (msg.action === 'setHealth') {
            game.health = msg.health;
            game.healthTotal = msg.healthTotal;
        } else if (msg.action === 'remove') {
            delete game.objects[msg.id];
        } else {
            console.log(msg);
        }
    }
    render();
};

document.onkeydown = function(event) {
    if (event.key === 'ArrowUp') {
        send({action: 'move', dir: 'up'});
    } else if (event.key === 'ArrowRight') {
        send({action: 'move', dir: 'right'});
    } else if (event.key === 'ArrowDown') {
        send({action: 'move', dir: 'down'});
    } else if (event.key === 'ArrowLeft') {
        send({action: 'move', dir: 'left'});
    } else {
        return;
    }
    event.preventDefault();
};

var click = function() {
    var rect = $dpad.getBoundingClientRect();
    var x = (pointer.x - rect.x) / rect.width - 0.5;
    var y = (pointer.y - rect.y) / rect.height - 0.5;
    if (Math.abs(x) > Math.abs(y)) {
        send({action: 'move', dir: x > 0 ? 'right' : 'left'});
    } else {
        send({action: 'move', dir: y > 0 ? 'down' : 'up'});
    }
};

$dpad.addEventListener('pointerdown', event => {
    if (!pointer && (event.buttons & 1 || event.pointerType !== 'mouse')) {
        event.preventDefault();
        $dpad.setPointerCapture(event.pointerId);
        pointer = {
            id: event.pointerId,
            x: event.clientX,
            y: event.clientY,
            timeout: setTimeout(() => {
                click();
                pointer.timeout = setInterval(() => {
                    click();
                }, 40);
            }, 200),
        };
        click();
    }
});

$dpad.addEventListener('pointermove', event => {
    if (pointer && event.pointerId === pointer.id) {
        event.preventDefault();
        pointer.x = event.clientX;
        pointer.y = event.clientY;
    }
});

var pointerup = function(event) {
    if (pointer && event.pointerId === pointer.id) {
        clearTimeout(pointer.timeout);
        pointer = null;
    }
};

$dpad.addEventListener('pointerup', pointerup);
$dpad.addEventListener('pointercancel', pointerup);
