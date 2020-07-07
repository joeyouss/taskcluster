const path = require('path');
const os = require('os');
const util = require('util');
const rimraf = util.promisify(require('rimraf'));
const mkdirp = util.promisify(require('mkdirp'));
const {TaskGraph, Lock, ConsoleRenderer, LogRenderer} = require('console-taskgraph');
const {Build} = require('../build');
const generateVersionTasks = require('./version');
const generatePublishTasks = require('./tasks');
const taskcluster = require('taskcluster-client');
const {
  gitDescribe,
  REPO_ROOT,
} = require('../utils');

class Publish {
  constructor(cmdOptions) {
    this.cmdOptions = cmdOptions;

    this.baseDir = cmdOptions['baseDir'] || '/tmp/taskcluster-builder-build';
    this.logsDir = cmdOptions['logsDir'] || path.join(this.baseDir, 'logs');

    // The `yarn build` process is a subgraph of the publish taskgraph, with some
    // options "forced"
    this.build = new Build({
      ...cmdOptions,
      cache: false, // always build from scratch
    });
  }

  async generateTasks() {
    // try loading the secret into process.env
    if (process.env.TASKCLUSTER_PROXY_URL) {
      const secretName = `project/taskcluster/${this.cmdOptions.staging ? 'staging-' : ''}release`;
      console.log(`loading secrets from taskcluster secret ${secretName} via taskcluster-proxy`);
      const secrets = new taskcluster.Secrets({rootUrl: process.env.TASKCLUSTER_PROXY_URL});
      const {secret} = await secrets.get(secretName);

      for (let [name, value] of Object.entries(secret)) {
        console.log(`..found value for ${name}`);
        process.env[name] = value;
      }
    }

    const expectedVars = [];
    expectedVars.push('GH_TOKEN');
    if (!this.cmdOptions.staging) {
      expectedVars.push('NPM_TOKEN');
      expectedVars.push('PYPI_USERNAME');
      expectedVars.push('PYPI_PASSWORD');
      expectedVars.push('DOCKER_USERNAME');
      expectedVars.push('DOCKER_PASSWORD');
    }

    expectedVars.forEach(e => {
      if (!process.env[e]) {
        throw new Error(`$${e} is required`);
      }
    });

    let tasks = this.build.generateTasks('release');
    generateVersionTasks({
      tasks,
      cmdOptions: this.cmdOptions,
      credentials: {},
      baseDir: this.baseDir,
      logsDir: this.logsDir,
    });
    generatePublishTasks({
      tasks,
      cmdOptions: this.cmdOptions,
      credentials: {
        ghToken: process.env.GH_TOKEN,
        npmToken: process.env.NPM_TOKEN,
        pypiUsername: process.env.PYPI_USERNAME,
        pypiPassword: process.env.PYPI_PASSWORD,
        dockerUsername: process.env.DOCKER_USERNAME,
        dockerPassword: process.env.DOCKER_PASSWORD,
      },
      baseDir: this.baseDir,
      logsDir: this.logsDir,
    });

    return tasks;
  }

  async getVersionInfo() {
    if (this.cmdOptions.staging) {
      // for staging releases, we get the version from the staging-release/*
      // branch name, and use a fake revision
      const match = /staging-release\/v(\d+\.\d+\.\d+)$/.exec(this.cmdOptions.staging);
      if (!match) {
        throw new Error(`Staging releases must have branches named 'staging-release/vX.Y.Z'; got ${this.cmdOptions.staging}`);
      }
      const version = match[1];

      return {
        'release-version': version,
        'release-revision': '9999999999999999999999999999999999999999',
      };
    } else {
      const {gitDescription, revision} = await gitDescribe({
        dir: REPO_ROOT,
      });

      if (!gitDescription.match(/^v\d+\.\d+\.\d+$/)) {
        throw new Error(`Can only publish releases from git revisions with tags of the form vX.Y.Z, not ${gitDescription}`);
      }

      return {
        'release-version': gitDescription.slice(1),
        'release-revision': revision,
      };
    }
  }

  async run() {
    // --staging implies --no-push
    if (this.cmdOptions.staging) {
      this.cmdOptions.push = false;
    }

    if (!this.cmdOptions.cache) {
      await rimraf(this.baseDir);
    }
    await mkdirp(this.baseDir);

    await rimraf(this.logsDir);
    await mkdirp(this.logsDir);

    let tasks = await this.generateTasks();

    const taskgraph = new TaskGraph(tasks, {
      locks: {
        // limit ourselves to one docker process per CPU
        docker: new Lock(os.cpus().length),
        // and let's be sane about how many git clones we do..
        git: new Lock(8),
      },
      target: 'target-publish',
      renderer: process.stdout.isTTY ?
        new ConsoleRenderer({elideCompleted: true}) :
        new LogRenderer(),
    });
    if (this.cmdOptions.dryRun) {
      console.log('Dry run successful.');
      return;
    }
    const context = await taskgraph.run({
      ...await this.getVersionInfo(),
    });

    console.log(`Release version: ${context['release-version']}`);
    console.log(`Release docker image: ${context['monoimage-docker-image']}`);
    console.log(`GitHub release: ${context['github-release']}`);
  }
}

const main = async (options) => {
  const publish = new Publish(options);
  await publish.run();
};

module.exports = {main, Publish};
