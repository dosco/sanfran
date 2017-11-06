module.exports = function (req, res) {
  res.send(`Hello ${req.query.name || 'World'}`);
};
