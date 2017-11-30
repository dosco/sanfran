'use strict';

const os = require('os');
const fs = require('fs');
const express = require('express')
const app = express();

const codepath = "/shared/func/function.js";
const port = 8081;

let func;

app.get('/api/ping', function (req, res) {
  try {
    res.status(200).send({
      load_avg: os.loadavg(),
      free_mem: (os.freemem() / os.totalmem()) * 100,
    });
  } catch(e) {
    console.error(`error fetching metrics: ${e}`);
    res.status(500).send(JSON.stringify(e));
    return;
  }
});

app.get('/api/activate', function (req, res) {
  try {
    delete require.cache[codepath];
    func = require(codepath);

    res.status(200).send();
  } catch(e) {
    console.error(`user code load error: ${e}`);
    res.status(500).send(JSON.stringify(e));
    return;
  }
});

app.all('/', function (req, res) {
  if (!func) {
    res.status(500).send("no function defined");
    return;
  }
  try {
    func(req, res);
  } catch (e) {
    console.error(`function error: ${e}`);
    res.status(500).send(JSON.stringify(e));
  }
});

app.listen(port, function () {
  console.log(`app listening on port ${port}!`);
})