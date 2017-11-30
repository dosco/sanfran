'use strict';

const os = require('os');
const fs = require('fs');
const path = require('path');
const express = require('express')
const app = express();

const codePath = '/shared/func/';
const funcPath = path.join(codePath, '/function.js');

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
    if (fs.existsSync(codePath)) {
      clearRequireCache(fs.realpathSync(codePath));
      func = require(funcPath);
    }
    res.status(200).send('activated');
  } catch(e) {
    console.error(`user code load error: ${e}`);
    res.status(500).send(JSON.stringify(e));
  }
});

app.all('/*', function (req, res) {
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

app.listen(port, function () {
  console.log(`app listening on port ${port}!`);
})

function clearRequireCache(prefix) {
  for (var k in require.cache) {
    if (k.startsWith(prefix)) {
      delete require.cache[k];
    }
  }
}