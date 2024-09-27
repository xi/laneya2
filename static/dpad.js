export default function(handler) {
    var $dpad = document.querySelector('#dpad');
    var pointer = null;

    var click = function() {
        var rect = $dpad.getBoundingClientRect();
        var x = (pointer.x - rect.x) / rect.width - 0.5;
        var y = (pointer.y - rect.y) / rect.height - 0.5;
        if (Math.abs(x) > Math.abs(y)) {
            handler(x > 0 ? 'ArrowRight' : 'ArrowLeft');
        } else {
            handler(y > 0 ? 'ArrowDown' : 'ArrowUp');
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
}
