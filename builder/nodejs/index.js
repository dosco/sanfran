const fs = require('fs');
const detect = require('detect-import-require');
const filepath = require('filepath');
const childProcess = require('child_process');
const filename = process.argv[2];

var jsFile = filepath.create(filename);
var modules = detect(fs.readFileSync(jsFile.path, 'utf8'));

var packageJSON = {
  "name": "function",
  "version": "1.0.0",
  "description": "",
  "dependencies": {},
};

var cwd = jsFile.dir().path;
var pjFile = filepath.create(cwd, "package.json");
fs.writeFileSync(pjFile.path, JSON.stringify(packageJSON));

for (var i = 0; i < modules.length; i++) {
  var cmd = `npm install ${modules[i]} ${cwd} --save --no-progress`;
  childProcess.execSync(cmd, {
    cwd: cwd,
    stdio:[0,1,2],
  });
}




