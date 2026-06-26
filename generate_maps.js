const fs = require('fs');
const path = require('path');

const refDir = path.join(__dirname, 'refrence', 'ffxiv-ember-overlay', 'src');

// Read jobs.json
const jobsJson = JSON.parse(fs.readFileSync(path.join(refDir, 'data', 'game', 'jobs.json'), 'utf8'));

// Extract GameJobs from index.js
const indexJs = fs.readFileSync(path.join(refDir, 'constants', 'index.js'), 'utf8');
const gameJobsMatch = indexJs.match(/const GameJobs = (\{[\s\S]*?\});/);

let gameJobs = {};
if (gameJobsMatch) {
    // using eval to parse the object literal since it's not strictly JSON
    eval(`gameJobs = ${gameJobsMatch[1]}`);
}

const jobMap = {};
for (const [id, data] of Object.entries(jobsJson)) {
    const abbr = data.abbreviation;
    if (gameJobs[abbr] && gameJobs[abbr].Name_en) {
        jobMap[id] = gameJobs[abbr].Name_en;
    } else {
        jobMap[id] = abbr;
    }
}

// Read instances.json
const instancesJson = JSON.parse(fs.readFileSync(path.join(refDir, 'data', 'game', 'instances.json'), 'utf8'));
const zoneMap = {};
for (const [id, data] of Object.entries(instancesJson)) {
    if (data.zone_id && data.locales && data.locales.name && data.locales.name.en) {
        zoneMap[data.zone_id] = data.locales.name.en;
    }
}

// Generate mappings.go
let goCode = `package main

var JobMap = map[string]string{\n`;
for (const [id, name] of Object.entries(jobMap)) {
    goCode += `\t"${id}": "${name}",\n`;
}
goCode += `}\n\n`;

goCode += `var ZoneMap = map[int]string{\n`;
for (const [id, name] of Object.entries(zoneMap)) {
    const escapedName = name.replace(/"/g, '\\"');
    goCode += `\t${id}: "${escapedName}",\n`;
}
goCode += `}\n`;

fs.writeFileSync(path.join(__dirname, 'mappings.go'), goCode);
console.log('mappings.go generated successfully');
