var replace = require('replace-in-file');

var buildNumber = process.argv[2];

var files = [
    "./src/environments/environment.ts",
    "./src/environments/environment.prod.ts",
];

var buildNumberOptions = {
    files: files,
    from: /buildNumber:.*/g,
    to: "buildNumber: " + buildNumber + ",",
    allowEmptyPaths: false
};

var protoVersionOptions = {
    files: files,
    from: /protoVersion:.*/g,
    to: "protoVersion: " + buildNumber + ",",
    allowEmptyPaths: false
};

replace.sync(buildNumberOptions);
replace.sync(protoVersionOptions);
