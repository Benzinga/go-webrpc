import WebRPC from "webrpc";

const client = new WebRPC("ws://localhost:4321");
const logContainer = document.querySelector(".log-container");
const chatbox = document.querySelector(".chatbox");
const chatboxInput = document.querySelector(".chatbox-input");
const msgTemplate = document.querySelector("#msg-template").innerText;
const joinTemplate = document.querySelector("#join-template").innerText;
const partTemplate = document.querySelector("#part-template").innerText;

function appendLog(template, values) {
    const atBottomish = logContainer.scrollTop >= logContainer.scrollHeight - logContainer.clientHeight - 10;

    for (const [key, value] of Object.entries(values)) {
        template = template.replace(`{{${key}}}`, value);
    }

    logContainer.innerHTML += template;

    if (atBottomish) {
        logContainer.scrollTop = logContainer.scrollHeight;
    }
}

client.on('join', function handleMsg(ts, user, chan) {
    appendLog(joinTemplate, { user, chan });
});

client.on('part', function handleMsg(ts, user, chan) {
    appendLog(partTemplate, { user, chan });
});

client.on('msg', function handleMsg(ts, user, msg) {
    appendLog(msgTemplate, { user, msg });
});

chatbox.addEventListener("submit", (event) => {
    event.preventDefault();
    client.emit("msg", "#welcome", chatboxInput.value);
    chatboxInput.value = "";
});
