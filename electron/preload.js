const { contextBridge } = require("electron");

contextBridge.exposeInMainWorld("soccerApp", {
  platform: process.platform,
  version: "1.0.0",
});
