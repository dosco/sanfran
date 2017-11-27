var express = require('express');
var hello = require('./hello.js');

var app = express();
app.get('/', hello);

app.listen(4000, function () {
  console.log('Example app listening on port 4000!');
});
