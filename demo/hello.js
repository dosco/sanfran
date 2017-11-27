var markdown = require( "markdown" ).markdown;

module.exports = function (req, res) {
  res.send( markdown.toHTML(`Hello *${req.query.name || 'World'}*`) );
};
