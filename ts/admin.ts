import { ThemeManager } from "./modules/theme.js";
import { lang, LangFile, loadLangSelector } from "./modules/lang.js";
import { Modal } from "./modules/modal.js";
import { TabManager, isPageEventBindable, isNavigatable } from "./modules/tabs.js";
import { DOMInviteList, createInvite } from "./modules/invites.js";
import { accountsList } from "./modules/accounts.js";
import { settingsList } from "./modules/settings.js";
import { activityList } from "./modules/activity.js";
import { ProfileEditor, reloadProfileNames } from "./modules/profiles.js";
import { _get, _post, notificationBox, whichAnimationEvent, bindManualDropdowns } from "./modules/common.js";
import { Updater } from "./modules/update.js";
import { Login } from "./modules/login.js";
import { setupTooltips } from "./modules/ui.js";

declare var window: GlobalWindow;

setupTooltips();

const theme = new ThemeManager(document.getElementById("button-theme"));

window.lang = new lang(window.langFile as LangFile);
loadLangSelector("admin");
// _get(`/lang/admin/${window.language}.json`, null, (req: XMLHttpRequest) => {
//     if (req.readyState == 4 && req.status == 200) {
//         langLoaded = true;
//         window.lang = new lang(req.response as LangFile);
//     }
// });

window.animationEvent = whichAnimationEvent();

window.token = "";

window.availableProfiles = window.availableProfiles || [];

// load modals
(() => {
    window.modals = {} as Modals;

    window.modals.login = new Modal(document.getElementById("modal-login"), true);

    window.modals.addUser = new Modal(document.getElementById("modal-add-user"));

    window.modals.about = new Modal(document.getElementById("modal-about"));
    (document.getElementById("setting-about") as HTMLSpanElement).onclick = window.modals.about.toggle;

    window.modals.modifyUser = new Modal(document.getElementById("modal-modify-user"));

    window.modals.deleteUser = new Modal(document.getElementById("modal-delete-user"));

    window.modals.settingsRestart = new Modal(document.getElementById("modal-restart"));

    window.modals.settingsRefresh = new Modal(document.getElementById("modal-refresh"));

    window.modals.ombiProfile = new Modal(document.getElementById("modal-ombi-profile"));
    document.getElementById("form-ombi-defaults").addEventListener("submit", window.modals.ombiProfile.close);

    window.modals.jellyseerrProfile = new Modal(document.getElementById("modal-jellyseerr-profile"));
    document
        .getElementById("form-jellyseerr-defaults")
        .addEventListener("submit", window.modals.jellyseerrProfile.close);

    window.modals.profiles = new Modal(document.getElementById("modal-user-profiles"));

    window.modals.addProfile = new Modal(document.getElementById("modal-add-profile"));

    window.modals.editProfile = new Modal(document.getElementById("modal-edit-profile"));

    window.modals.announce = new Modal(document.getElementById("modal-announce"));

    window.modals.editor = new Modal(document.getElementById("modal-editor"));

    window.modals.customizeEmails = new Modal(document.getElementById("modal-customize"));

    window.modals.extendExpiry = new Modal(document.getElementById("modal-extend-expiry"));

    window.modals.updateInfo = new Modal(document.getElementById("modal-update"));

    window.modals.matrix = new Modal(document.getElementById("modal-matrix"));

    window.modals.logs = new Modal(document.getElementById("modal-logs"));

    window.modals.tasks = new Modal(document.getElementById("modal-tasks"));

    window.modals.backedUp = new Modal(document.getElementById("modal-backed-up"));

    window.modals.backups = new Modal(document.getElementById("modal-backups"));

    if (window.telegramEnabled) {
        window.modals.telegram = new Modal(document.getElementById("modal-telegram"));
    }

    if (window.discordEnabled) {
        window.modals.discord = new Modal(document.getElementById("modal-discord"));
    }

    if (window.linkResetEnabled) {
        window.modals.sendPWR = new Modal(document.getElementById("modal-send-pwr"));
    }

    if (window.referralsEnabled) {
        window.modals.enableReferralsUser = new Modal(document.getElementById("modal-enable-referrals-user"));
        window.modals.enableReferralsProfile = new Modal(document.getElementById("modal-enable-referrals-profile"));
    }
})();

// Make the navbar horizontally scrollable by dragging (with mouse)
// doesn't work incredibly well so disabled.
/*[...document.getElementsByClassName("horizontally-scrollable")].forEach((c: HTMLElement) => {
    c.classList.add("cursor-pointer");
    let down = false;
    let startX: number, scrollLeft: number;
    c.addEventListener("mousedown", (ev: MouseEvent) => {
        console.log("down");
        down = true;
        c.classList.add("active");
        startX = ev.pageX - c.offsetLeft;
        scrollLeft = c.scrollLeft;
    });
    const leave = () => {
        console.log("up");
        down = false;
        c.classList.remove("active");
    };
    c.addEventListener("mouseleave", leave);
    c.addEventListener("mouseup", leave);
    c.addEventListener("mousemove", (ev: MouseEvent) => {
        if (!down) return;
        const x = ev.pageX - c.offsetLeft;
        const walk = x - startX;
        c.scrollLeft = scrollLeft - walk;
    });
});*/

// tab content objects will register with this independently, so initialise now
window.tabs = new TabManager();

var inviteCreator = new createInvite();

var accounts = new accountsList();

var activity = new activityList();

window.invites = new DOMInviteList();

var settings = new settingsList();

var profiles = new ProfileEditor();

window.notifications = new notificationBox(document.getElementById("notification-box") as HTMLDivElement, 5);

// only use a navigatable URL once
let navigated = false;

// load tabs
const tabs: { id: string; url: string; reloader: () => void; unloader?: () => void }[] = [];
[window.invites, accounts, activity, settings].forEach((p: AsTab) => {
    let t: { id: string; url: string; reloader: (previous?: AsTab) => void; unloader?: () => void } = {
        id: p.tabName,
        url: p.pagePath,
        reloader: (previous: AsTab) => {
            if (isPageEventBindable(p)) p.bindPageEvents();
            if (!navigated && isNavigatable(p) && p.isURL()) {
                navigated = true;
                p.navigate();
            } else {
                if (navigated && previous && isNavigatable(previous)) {
                    // Clear the query param, as it was likely for a different page
                    previous.clearURL();
                }
                p.reload(() => {});
            }
        },
    };
    if (isPageEventBindable(p)) t.unloader = p.unbindPageEvents;
    tabs.push(t);
    window.tabs.addTab(
        t.id,
        window.pages.Base + window.pages.Admin + "/" + t.url,
        p,
        null,
        t.reloader,
        t.unloader || null,
    );
});

let matchedTab = false;
for (const tab of tabs) {
    if (window.location.pathname.startsWith(window.pages.Base + window.pages.Current + "/" + tab.url)) {
        window.tabs.switch(tab.url, true);
        matchedTab = true;
    }
}
// Default tab
if (!matchedTab) {
    window.tabs.switch("", true);
}

const login = new Login(window.modals.login as Modal, "/", window.loginAppearance);

const setJellyfinStatus = (state: "online" | "offline" | "checking") => {
    const wrap = document.querySelector(".of-jf-status") as HTMLElement;
    const label = document.querySelector(".of-status-label") as HTMLElement;
    if (!wrap || !label) return;
    wrap.classList.remove("offline", "checking");
    if (state === "online") {
        label.textContent = "Jellyfin Online";
    } else if (state === "offline") {
        wrap.classList.add("offline");
        label.textContent = "Jellyfin Offline";
    } else {
        wrap.classList.add("checking");
        label.textContent = "Checking...";
    }
};

const checkJellyfinStatus = () => {
    _get("/users/count", null, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        setJellyfinStatus(req.status === 200 ? "online" : "offline");
    });
};

const activityTypeIcon = (t: string): string => {
    switch (t) {
        case "creation":
        case "accountCreation":
            return "ri-user-add-line";
        case "deletion":
            return "ri-user-unfollow-line";
        case "disabled":
            return "ri-user-forbid-line";
        case "enabled":
            return "ri-user-follow-line";
        case "contactLinked":
            return "ri-link";
        case "contactUnlinked":
            return "ri-link-unlink";
        case "changePassword":
        case "resetPassword":
            return "ri-lock-password-line";
        default:
            return "ri-history-line";
    }
};

const renderActivityWidget = () => {
    const list = document.getElementById("of-activity-widget-list");
    if (!list) return;
    _post(
        "/activity",
        {
            page: 0,
            limit: 5,
            sortByField: "time",
            ascending: false,
            searchTerms: [],
            queries: [],
        },
        (req: XMLHttpRequest) => {
            if (req.readyState != 4) return;
            if (req.status != 200) return;
            const acts = (req.response?.activities || []) as Array<{
                type: string;
                username: string;
                source_username: string;
                value: string;
                time: number;
            }>;
            if (!acts.length) {
                list.innerHTML = `<span class="opacity-60 italic">No activity yet.</span>`;
                return;
            }
            list.innerHTML = "";
            const fmt = (ts: number) => {
                const d = new Date(ts * 1000);
                const diff = (Date.now() - d.getTime()) / 1000;
                if (diff < 60) return `${Math.floor(diff)}s ago`;
                if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
                if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
                return `${Math.floor(diff / 86400)}d ago`;
            };
            const esc = (s: string) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
            for (const a of acts) {
                const row = document.createElement("div");
                row.className = "flex flex-row items-center gap-2";
                const who = esc(a.username || a.source_username || "—");
                const type = esc(a.type);
                const value = a.value ? ` <span class="opacity-50">(${esc(a.value)})</span>` : "";
                row.innerHTML = `
                    <i class="${activityTypeIcon(a.type)} opacity-70"></i>
                    <span class="font-mono text-xs opacity-60 w-16">${fmt(a.time)}</span>
                    <span class="truncate"><strong>${who}</strong> · <span class="opacity-70">${type}</span>${value}</span>
                `;
                list.appendChild(row);
            }
        },
        true,
    );
};

// Render the Jellystat watch-stats widget using the same /users/announce-vars endpoint
// the announcement editor uses, so the data is always consistent with what users will see in emails.
const renderWatchWidget = () => {
    const widget = document.getElementById("of-watch-widget");
    const summary = document.getElementById("of-watch-widget-summary");
    const usersEl = document.getElementById("of-watch-widget-users");
    const titlesEl = document.getElementById("of-watch-widget-titles");
    if (!widget || !summary || !usersEl || !titlesEl) return;
    _get("/users/announce-vars", null, (req: XMLHttpRequest) => {
        if (req.readyState != 4) return;
        if (req.status != 200) {
            widget.classList.add("ui-hidden");
            return;
        }
        const vars = ((req.response as { vars: Record<string, string> })?.vars) || {};
        // If none of the Jellystat keys are present, just hide the widget.
        const hasStats = vars["watch_plays_30d"] || vars["watch_time_30d"] || vars["active_watchers_30d"] || vars["top_users_30d"];
        if (!hasStats) {
            widget.classList.add("ui-hidden");
            return;
        }
        widget.classList.remove("ui-hidden");
        const tile = (label: string, value: string, icon: string) => `
            <div class="card ~neutral @low flex flex-col gap-1 p-3">
                <span class="text-xs opacity-60 flex flex-row items-center gap-1"><i class="${icon}"></i>${label}</span>
                <span class="text-lg font-semibold">${value || "—"}</span>
            </div>`;
        summary.innerHTML =
            tile("Plays", vars["watch_plays_30d"] || "0", "ri-play-circle-line") +
            tile("Watch time", vars["watch_time_30d"] || "0m", "ri-time-line") +
            tile("Active watchers", vars["active_watchers_30d"] || "0", "ri-user-line") +
            tile("Members", vars["user_count"] || "0", "ri-group-line");
        // top_users_30d / top_titles_30d come back as markdown bullet lists; render as raw HTML
        // (Marked-style passthrough — the server-side stats endpoint already escapes user content).
        const mdToHTML = (md: string) => {
            if (!md) return `<span class="opacity-60 italic">No data</span>`;
            return md
                .split("\n")
                .map((line) => line.replace(/^[-*]\s*/, "").trim())
                .filter(Boolean)
                .map((line) => {
                    // Bold the part inside **...**
                    const html = line.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>");
                    return `<div class="truncate">• ${html}</div>`;
                })
                .join("");
        };
        usersEl.innerHTML = mdToHTML(vars["top_users_30d"] || "");
        titlesEl.innerHTML = mdToHTML(vars["top_titles_30d"] || "");
    });
};

const wireActivityWidgetLink = () => {
    const more = document.getElementById("of-activity-widget-more");
    if (more && !more.dataset.wired) {
        more.dataset.wired = "1";
        more.addEventListener("click", (e) => {
            e.preventDefault();
            (window as any).tabs?.switch?.("activity");
        });
    }
};

const sidebarNav = document.querySelector(".of-sidebar-nav") as HTMLElement | null;
const lockSidebar = () => sidebarNav?.classList.add("of-nav-locked");
const unlockSidebar = () => sidebarNav?.classList.remove("of-nav-locked");
lockSidebar();

login.onLogin = () => {
    unlockSidebar();
    window.updater = new Updater();
    // FIXME: Decide whether to autoload activity or not
    reloadProfileNames();
    checkJellyfinStatus();
    renderActivityWidget();
    renderWatchWidget();
    wireActivityWidgetLink();
    setInterval(() => {
        window.invites.reload();
        accounts.reloadIfNotInScroll();
        checkJellyfinStatus();
        renderActivityWidget();
    }, 30 * 1000);
    // Watch stats change slower; refresh on a longer interval to keep Jellystat load light.
    setInterval(renderWatchWidget, 5 * 60 * 1000);
    // Triggers pre and post funcs, even though we're already on that page
    window.tabs.switch(window.tabs.current);
};

bindManualDropdowns();

login.bindLogout(document.getElementById("logout-button"));

const showShortcutHelp = () => {
    let panel = document.getElementById("of-shortcut-help");
    if (!panel) {
        panel = document.createElement("div");
        panel.id = "of-shortcut-help";
        panel.className = "modal";
        panel.innerHTML = `
            <div class="card mx-auto my-[15%] w-11/12 sm:w-2/3 lg:w-1/3 flex flex-col gap-2">
                <div class="flex flex-row justify-between items-center">
                    <span class="heading mb-0">Keyboard shortcuts</span>
                    <span class="modal-close cursor-pointer text-2xl leading-none">&times;</span>
                </div>
                <table class="table">
                    <tbody>
                        <tr><td><kbd class="font-mono px-2 py-0.5 rounded bg-black/10 dark:bg-white/10">Cmd / Ctrl + K</kbd></td><td>Focus search</td></tr>
                        <tr><td><kbd class="font-mono px-2 py-0.5 rounded bg-black/10 dark:bg-white/10">Esc</kbd></td><td>Close any open modal</td></tr>
                        <tr><td><kbd class="font-mono px-2 py-0.5 rounded bg-black/10 dark:bg-white/10">?</kbd></td><td>Show this help</td></tr>
                    </tbody>
                </table>
            </div>
        `;
        document.body.appendChild(panel);
        const close = () => panel?.classList.remove("block", "animate-fade-in");
        panel.querySelector(".modal-close")?.addEventListener("click", close);
        panel.addEventListener("click", (e) => { if (e.target === panel) close(); });
    }
    panel.classList.add("block", "animate-fade-in");
};

document.addEventListener("keydown", (e: KeyboardEvent) => {
    const target = e.target as HTMLElement;
    const isTyping = target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable);

    // Cmd/Ctrl+K — focus the current tab's search input
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        const current = (window as any).tabs?.current;
        const search = current ? (document.getElementById(current + "-search") as HTMLInputElement | null) : null;
        if (search) {
            e.preventDefault();
            search.focus();
            search.select();
        }
        return;
    }

    if (isTyping) return;

    // Shift+/ produces "?" — show shortcut help (Esc handled by Modal class itself)
    if (e.key === "?" && !e.metaKey && !e.ctrlKey && !e.altKey) {
        e.preventDefault();
        showShortcutHelp();
    }
});

login.login("", "");
