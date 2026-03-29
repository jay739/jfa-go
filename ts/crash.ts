import { toClipboard } from "./modules/common.js";

const buttonNormal = document.getElementById("button-log-normal") as HTMLInputElement;
const buttonSanitized = document.getElementById("button-log-sanitized") as HTMLInputElement;

const logNormal = document.getElementById("log-normal") as HTMLInputElement;
const logSanitized = document.getElementById("log-sanitized") as HTMLInputElement;

const buttonChange = (type: string) => {
    if (type == "normal") {
        logSanitized.classList.add("ui-hidden");
        logNormal.classList.remove("ui-hidden");
        buttonNormal.classList.add("@high");
        buttonNormal.classList.remove("@low");
        buttonSanitized.classList.add("@low");
        buttonSanitized.classList.remove("@high");
    } else {
        logNormal.classList.add("ui-hidden");
        logSanitized.classList.remove("ui-hidden");
        buttonSanitized.classList.add("@high");
        buttonSanitized.classList.remove("@low");
        buttonNormal.classList.add("@low");
        buttonNormal.classList.remove("@high");
    }
};
buttonNormal.onclick = () => buttonChange("normal");
buttonSanitized.onclick = () => buttonChange("sanitized");

const copyButton = document.getElementById("copy-log") as HTMLSpanElement;
const copyLabel = document.body.dataset.copyLabel || "Copy";
const copiedLabel = document.body.dataset.copiedLabel || "Copied.";
copyButton.onclick = () => {
    if (logSanitized.classList.contains("ui-hidden")) {
        toClipboard("```\n" + logNormal.textContent + "```");
    } else {
        toClipboard("```\n" + logSanitized.textContent + "```");
    }
    copyButton.textContent = copiedLabel;
    copyButton.classList.add("~positive");
    copyButton.classList.remove("~urge");
    setTimeout(() => {
        copyButton.textContent = copyLabel;
        copyButton.classList.add("~urge");
        copyButton.classList.remove("~positive");
    }, 1500);
};
