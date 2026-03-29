declare var window: GlobalWindow;

export class Modal implements Modal {
    modal: HTMLElement;
    closeButton: HTMLSpanElement | null;
    openEvent: CustomEvent;
    closeEvent: CustomEvent;
    private _activeBeforeOpen: HTMLElement | null = null;
    private _escapeListener = ((e: KeyboardEvent) => {
        if (e.key === "Escape") {
            e.preventDefault();
            this.close();
        }
    }).bind(this);

    constructor(modal: HTMLElement, important: boolean = false) {
        this.modal = modal;
        this.openEvent = new CustomEvent("modal-open-" + modal.id);
        this.closeEvent = new CustomEvent("modal-close-" + modal.id);
        const closeButton = this.modal.querySelector("span.modal-close");
        if (closeButton !== null) {
            this.closeButton = closeButton as HTMLSpanElement;
            this.closeButton.onclick = this.close;
        } else {
            this.closeButton = null;
        }
        if (!important) {
            window.addEventListener("click", (event: Event) => {
                if (event.target == this.modal) {
                    this.close();
                }
            });
        }
    }
    close = (event?: Event, noDispatch?: boolean) => {
        // If we don't check we can mess up a closed modal.
        if (!this.modal.classList.contains("block") && !this.modal.classList.contains("animate-fade-in")) return;
        if (event) {
            event.preventDefault();
        }
        document.removeEventListener("keydown", this._escapeListener);
        this.modal.classList.add("animate-fade-out");
        this.modal.classList.remove("animate-fade-in");
        const modal = this.modal;
        const returnFocus = this._activeBeforeOpen;
        this._activeBeforeOpen = null;
        const listenerFunc = () => {
            modal.classList.remove("block");
            modal.classList.remove("animate-fade-out");
            modal.removeEventListener(window.animationEvent, listenerFunc);
            if (!noDispatch) document.dispatchEvent(this.closeEvent);
            returnFocus?.focus?.();
        };
        this.modal.addEventListener(window.animationEvent, listenerFunc, false);
    };

    set onopen(f: () => void) {
        document.addEventListener("modal-open-" + this.modal.id, f);
    }
    set onclose(f: () => void) {
        document.addEventListener("modal-close-" + this.modal.id, f);
    }

    private _focusFirstControl = () => {
        const sel =
            'button:not([disabled]), [href], input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';
        const focusable = this.modal.querySelectorAll<HTMLElement>(sel);
        const first = focusable[0];
        if (first) first.focus();
        else this.closeButton?.focus();
    };

    show = () => {
        if (this.modal.classList.contains("animate-fade-in")) return;
        this._activeBeforeOpen = document.activeElement instanceof HTMLElement ? document.activeElement : null;
        this.modal.setAttribute("role", "dialog");
        this.modal.setAttribute("aria-modal", "true");
        this.modal.classList.add("block", "animate-fade-in");
        document.addEventListener("keydown", this._escapeListener);
        document.dispatchEvent(this.openEvent);
        queueMicrotask(() => this._focusFirstControl());
    };
    toggle = () => {
        if (this.modal.classList.contains("animate-fade-in")) {
            this.close();
        } else {
            this.show();
        }
    };

    asElement = () => {
        return this.modal;
    };
}
