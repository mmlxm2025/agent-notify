#!/usr/bin/env node
const fs = require('node:fs');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { getPlatformTarget } = require('../lib/platform');
const { extractSemver, compareVersions } = require('../lib/version');
const { getInstalledBinaryPath, TMP_DIR } = require('../lib/constants');
const { buildAssetName, buildDownloadUrl } = require('../lib/release');
const { downloadToFile } = require('../lib/download');
const { installFromArchive } = require('../lib/install');
const { runBinary } = require('../lib/run');
const { clearQuarantine, adHocSign, NOTIFIER_BUNDLE } = require('../lib/notifier');

function getDesiredVersion() {
  return require('../package.json').version;
}

function getInstalledVersion(binaryPath) {
  const result = spawnSync(binaryPath, ['--version'], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  if (result.error || result.status !== 0) {
    return null;
  }

  return extractSemver(result.stdout);
}

async function downloadAndInstall(version, target, binaryPath) {
  fs.mkdirSync(TMP_DIR, { recursive: true });

  const assetName = buildAssetName(version, target);
  const archivePath = path.join(TMP_DIR, assetName);
  const binaryNameInArchive = `agent-notify-v${version}-${target.goos}-${target.goarch}${target.ext}`;

  await downloadToFile(buildDownloadUrl(version, assetName), archivePath);
  const installed = installFromArchive({
    archivePath,
    installDir: path.dirname(binaryPath),
    binaryNameInArchive,
    finalBinaryName: path.basename(binaryPath),
  });

  // macOS：terminal-notifier.app 已随 tar.gz 解压到 installDir（见 install.js），
  // 此处只做 quarantine 清除与 ad-hoc 签名（点击跳转依赖）。失败仅警告，不阻断主流程。
  if (process.platform === 'darwin') {
    const bundlePath = path.join(path.dirname(binaryPath), NOTIFIER_BUNDLE);
    if (fs.existsSync(bundlePath)) {
      try { clearQuarantine(bundlePath); } catch {}
      try { adHocSign(bundlePath); } catch {}
    }
  }

  return installed;
}

async function main(argv, deps = {}) {
  const desiredVersion = (deps.getDesiredVersion || getDesiredVersion)();
  const target = (deps.getPlatformTarget || getPlatformTarget)();
  const binaryPath = (deps.getInstalledBinaryPath || getInstalledBinaryPath)(target);
  const pathExists = deps.pathExists || fs.existsSync;
  const getVersion = deps.getInstalledVersion || getInstalledVersion;
  const compare = deps.compareVersions || compareVersions;
  const install = deps.downloadAndInstall || ((version, releaseTarget) => downloadAndInstall(version, releaseTarget, binaryPath));
  const run = deps.runBinary || runBinary;
  const warn = deps.warn || ((message) => console.warn(message));

  let installedVersion = null;
  const hasInstalledBinary = pathExists(binaryPath);
  if (hasInstalledBinary) {
    installedVersion = getVersion(binaryPath);
  }
  const canFallbackToInstalledBinary = Boolean(installedVersion);

  const needsInstall = !hasInstalledBinary || !installedVersion || compare(installedVersion, desiredVersion) < 0;

  if (needsInstall) {
    try {
      await install(desiredVersion, target, binaryPath);
    } catch (error) {
      if (!canFallbackToInstalledBinary) {
        throw error;
      }
      warn(`failed to update agent-notify: ${error.message}`);
    }
  }

  return run(binaryPath, argv);
}

if (require.main === module) {
  main(process.argv.slice(2))
    .then((code) => {
      process.exitCode = code;
    })
    .catch((error) => {
      console.error(error.message);
      process.exitCode = 1;
    });
}

module.exports = {
  main,
  downloadAndInstall,
  getDesiredVersion,
  getInstalledVersion,
};
