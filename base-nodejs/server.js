'use strict';

const os = require('os');
const fs = require('fs');
const path = require('path');
const express = require('express');
const app = express();

const codePath = '/shared/func/';
const funcPath = path.join(codePath, '/function.js');
const envPath = path.join(codePath, '/.env');

const port = 8081;

require('dotenv').config({path: envPath})

const func = (() => {
  try {
    let hrstart = process.hrtime();
    let v =  require(funcPath);
    let hrend = process.hrtime(hrstart);
    console.info("Function load time: %ds %dms", hrend[0], hrend[1]/1000000);
    return v;
  }
  catch(e) { console.error(`user code load error: ${e}`); }
})();

app.get('/api/ping', (req, res) => {
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
    res.status(200).send('activated');
    process.exit();
});

app.all('/*', (req, res) => {
  const qs = req.originalUrl.indexOf('?');
  const _url = (qs === -1)
    ? req.originalUrl : req.originalUrl.substring(0, qs);

  if (!func) {
    res.status(500).send("no function defined");
    return;
  }

  try {
    if (typeof func === 'object') {
      const funcName = _url.substring(1).split('/', 2)[0];
      if (funcName in func) {
        func[funcName](req, res);
        return
      }
      res.status(404).send('Not found')
    } else {
      func(req, res);
    }
  } catch (e) {
    console.error(`function error: ${e}`);
    res.status(500).send(JSON.stringify(e));
  }
});

process.on('exit', (code) => {
  console.log(`Existing with code ${code}`);
});

const server = app.listen(port, () => {
  console.log(`app listening on port ${port}!`);
})