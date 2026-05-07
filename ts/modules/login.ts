import { Modal } from "../modules/modal.js";
import { toggleLoader, _post, unicodeB64Encode, formatApiFailure } from "../modules/common.js";

declare var window: GlobalWindow;

export class Login {
    loggedIn: boolean = false;
    private _modal: Modal;
    private _form: HTMLFormElement;
    private _url: string;
    private _endpoint: string;
    private _onLogin: (username: string, password: string) => void;
    private _logoutButton: HTMLElement = null;
    private _wall: HTMLElement;
    private _hasOpacityWall: boolean = false;

    constructor(modal: Modal, endpoint: string, appearance: string) {
        this._endpoint = endpoint;
        this._url = window.pages.Base + endpoint;
        if (this._url[this._url.length - 1] != "/") this._url += "/";

        this._modal = modal;
        if (appearance == "opaque") {
            this._hasOpacityWall = true;
            this._wall = document.createElement("div");
            this._wall.classList.add("wall");
            this._modal.asElement().parentElement.appendChild(this._wall);
        }
        this._form = this._modal.asElement().querySelector(".form-login") as HTMLFormElement;
        this._form.onsubmit = (event: SubmitEvent) => {
            event.preventDefault();
            const button = (event.target as HTMLElement).querySelector(".submit") as HTMLSpanElement;
            const username = (document.getElementById("login-user") as HTMLInputElement).value;
            const password = (document.getElementById("login-password") as HTMLInputElement).value;
            if (!username || !password) {
                window.notifications.customError("loginError", window.lang.notif("errorLoginBlank"));
                return;
            }
            toggleLoader(button);
            this.login(username, password, () => toggleLoader(button));
        };
    }

    private _refreshKey = () => "omnifin_refresh_" + this._endpoint;

    bindLogout = (button: HTMLElement) => {
        this._logoutButton = button;
        this._logoutButton.classList.add("ui-hidden");
        const logoutFunc = (url: string, tryAgain: boolean) => {
            _post(
                url + "logout",
                null,
                (req: XMLHttpRequest): boolean => {
                    if (req.readyState == 4 && req.status == 200) {
                        window.token = "";
                        localStorage.removeItem(this._refreshKey());
                        location.reload();
                        return false;
                    }
                },
                false,
                (req: XMLHttpRequest) => {
                    if (req.readyState == 4 && req.status == 404 && tryAgain) {
                        console.warn("logout failed, trying without URL Base...");
                        logoutFunc(this._endpoint, false);
                    }
                },
            );
        };
        this._logoutButton.onclick = () => logoutFunc(this._url, true);
    };

    get onLogin() {
        return this._onLogin;
    }
    set onLogin(f: (username: string, password: string) => void) {
        this._onLogin = f;
    }

    login = (username: string, password: string, run?: (state?: number) => void) => {
        const req = new XMLHttpRequest();
        req.responseType = "json";
        const refresh = username == "" && password == "";
        req.open("GET", this._url + (refresh ? "token/refresh" : "token/login"), true);
        if (!refresh) {
            req.setRequestHeader("Authorization", "Basic " + unicodeB64Encode(username + ":" + password));
        } else {
            // On refresh, fall back to a stored refresh JWT (Bearer header) when the HttpOnly
            // cookie isn't sent (e.g. cross-context navigation, third-party-cookie blocking).
            const storedRefresh = localStorage.getItem(this._refreshKey());
            if (storedRefresh) {
                req.setRequestHeader("Authorization", "Bearer " + storedRefresh);
            }
        }
        req.onreadystatechange = ((req: XMLHttpRequest, _: Event): any => {
            if (req.readyState == 4) {
                if (req.status != 200) {
                    if (refresh) {
                        localStorage.removeItem(this._refreshKey());
                    }
                    if (!refresh) {
                        window.notifications.customError(
                            "loginError",
                            formatApiFailure(req, window.lang.notif("errorUnknown")),
                        );
                    } else {
                        this._modal.show();
                    }
                } else {
                    const data = req.response;
                    window.token = data["token"];
                    if (data["refresh"]) {
                        localStorage.setItem(this._refreshKey(), data["refresh"]);
                    }
                    this.loggedIn = true;
                    if (this._onLogin) {
                        this._onLogin(username, password);
                    }
                    if (this._hasOpacityWall) this._wall.remove();
                    this._modal.close();
                    if (this._logoutButton != null) this._logoutButton.classList.remove("ui-hidden");
                }
                if (run) {
                    run(+req.status);
                }
            }
        }).bind(this, req);
        req.send();
    };
}
