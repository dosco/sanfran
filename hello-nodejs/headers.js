module.exports = function (req, res) {
  res.send(JSON.stringify([req.headers, req.query], null, 2));
};
