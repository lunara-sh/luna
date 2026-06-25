<?php
session_start();

define('LUNABASE_VERSION', '1.0.0');

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────
function makeDsn($driver, $host, $port, $dbname)
{
  if ($driver === 'pgsql') return "pgsql:host={$host};port={$port};dbname={$dbname}";
  return "mysql:host={$host};port={$port};dbname={$dbname};charset=utf8mb4";
}

function getConnection()
{
  if (!isset($_SESSION['db'])) return null;
  $db = $_SESSION['db'];
  try {
    $pdo = new PDO(
      makeDsn($db['driver'], $db['host'], $db['port'], $db['dbname']),
      $db['user'],
      $db['password'],
      [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION, PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC]
    );
    return $pdo;
  } catch (Exception $e) {
    return null;
  }
}

function getDatabases($pdo, $driver)
{
  if ($driver === 'pgsql') {
    return array_column($pdo->query("SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")->fetchAll(), 'datname');
  }
  return array_column($pdo->query("SHOW DATABASES")->fetchAll(), 'Database');
}

function getTables($pdo, $driver)
{
  if ($driver === 'pgsql') {
    return array_column($pdo->query("SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_catalog=current_database() ORDER BY table_name")->fetchAll(), 'table_name');
  }
  $db = array_column($pdo->query("SHOW TABLES")->fetchAll(), 0);
  return $db;
}

function getRowCount($pdo, $table, $driver)
{
  try {
    return $pdo->query("SELECT COUNT(*) FROM " . qi($table, $driver))->fetchColumn();
  } catch (Exception $e) {
    return 0;
  }
}

function qi($name, $driver)
{
  if ($driver === 'pgsql') return '"' . str_replace('"', '""', $name) . '"';
  return '`' . str_replace('`', '``', $name) . '`';
}

function safeDb($name)
{
  return preg_replace('/[^a-zA-Z0-9_]/', '', $name);
}

// ─────────────────────────────────────────────
// Actions
// ─────────────────────────────────────────────
$action  = $_GET['action'] ?? 'login';
$error   = '';
$success = '';

// LOGIN
if ($action === 'login' && $_SERVER['REQUEST_METHOD'] === 'POST') {
  $driver   = $_POST['driver']   ?? 'pgsql';
  $host     = $_POST['host']     ?? 'localhost';
  $port     = $_POST['port']     ?? ($driver === 'pgsql' ? '5432' : '3306');
  $user     = $_POST['user']     ?? '';
  $password = $_POST['password'] ?? '';
  $dbname   = trim($_POST['dbname'] ?? '');
  $defaultDb = $driver === 'pgsql' ? 'postgres' : 'mysql';
  $targetDb  = $dbname ?: $defaultDb;
  $opts = [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION];

  try {
    new PDO(makeDsn($driver, $host, $port, $targetDb), $user, $password, $opts);
    $_SESSION['db'] = compact('driver', 'host', 'port', 'user', 'password') + ['dbname' => $targetDb];
    header('Location: ?action=dashboard');
    exit;
  } catch (Exception $e) {
    try {
      $pdoDef = new PDO(makeDsn($driver, $host, $port, $defaultDb), $user, $password, $opts);
      if ($targetDb !== $defaultDb) {
        $safe = safeDb($targetDb);
        if ($safe) {
          if ($driver === 'pgsql') $pdoDef->exec("CREATE DATABASE \"$safe\"");
          else $pdoDef->exec("CREATE DATABASE `$safe`");
          new PDO(makeDsn($driver, $host, $port, $safe), $user, $password, $opts);
          $_SESSION['db'] = compact('driver', 'host', 'port', 'user', 'password') + ['dbname' => $safe];
          header('Location: ?action=dashboard&created_db=' . urlencode($safe));
          exit;
        }
      }
      $_SESSION['db'] = compact('driver', 'host', 'port', 'user', 'password') + ['dbname' => $defaultDb];
      header('Location: ?action=dashboard');
      exit;
    } catch (Exception $e2) {
      $error = $e2->getMessage();
    }
  }
}

// LOGOUT
if ($action === 'logout') {
  session_destroy();
  header('Location: ?action=login');
  exit;
}

// SWITCH DB
if ($action === 'switch_db' && $_SERVER['REQUEST_METHOD'] === 'POST' && isset($_SESSION['db'])) {
  $_SESSION['db']['dbname'] = $_POST['dbname'];
  header('Location: ?action=dashboard');
  exit;
}

// CREATE DATABASE
if ($action === 'create_database' && $_SERVER['REQUEST_METHOD'] === 'POST' && isset($_SESSION['db'])) {
  $newDb  = safeDb(trim($_POST['dbname'] ?? ''));
  $driver = $_SESSION['db']['driver'];
  if ($newDb) {
    try {
      $pdo_tmp = getConnection();
      if ($driver === 'pgsql') $pdo_tmp->exec("CREATE DATABASE \"$newDb\"");
      else $pdo_tmp->exec("CREATE DATABASE `$newDb`");
      $_SESSION['db']['dbname'] = $newDb;
      header('Location: ?action=dashboard&created_db=' . urlencode($newDb));
      exit;
    } catch (Exception $e) {
      $error = $e->getMessage();
    }
  } else {
    $error = 'Nama database tidak valid.';
  }
}

// DROP DATABASE
if ($action === 'drop_database' && $_SERVER['REQUEST_METHOD'] === 'POST' && isset($_SESSION['db'])) {
  $dropDb  = safeDb(trim($_POST['dbname'] ?? ''));
  $driver  = $_SESSION['db']['driver'];
  $defaultDb = $driver === 'pgsql' ? 'postgres' : 'mysql';
  $protected = ['postgres', 'mysql', 'information_schema', 'performance_schema', 'sys'];
  if (in_array($dropDb, $protected)) {
    $error = 'Database ini tidak bisa dihapus.';
  } elseif ($dropDb) {
    try {
      $prev = $_SESSION['db']['dbname'];
      $_SESSION['db']['dbname'] = $defaultDb;
      $pdo_tmp = getConnection();
      if ($driver === 'pgsql') {
        $pdo_tmp->exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$dropDb'");
        $pdo_tmp->exec("DROP DATABASE \"$dropDb\"");
      } else {
        $pdo_tmp->exec("DROP DATABASE `$dropDb`");
      }
      header('Location: ?action=dashboard&dropped_db=' . urlencode($dropDb));
      exit;
    } catch (Exception $e) {
      $_SESSION['db']['dbname'] = $prev;
      $error = $e->getMessage();
    }
  }
}

// CREATE TABLE
if ($action === 'create_table' && $_SERVER['REQUEST_METHOD'] === 'POST' && isset($_SESSION['db'])) {
  $driver = $_SESSION['db']['driver'];
  $tname  = trim($_POST['table_name'] ?? '');
  $cols   = $_POST['columns'] ?? [];
  if (!$tname) {
    $error = 'Nama table tidak boleh kosong.';
  } elseif (empty($cols)) {
    $error = 'Minimal satu kolom.';
  } else {
    $defs = [];
    foreach ($cols as $col) {
      $cname = trim($col['name'] ?? '');
      if (!$cname) continue;
      $ctype = $col['type'] ?? 'VARCHAR(255)';
      $cauto = isset($col['auto']);
      $cpk   = isset($col['primary']);
      $cnull = isset($col['nullable']) ? '' : 'NOT NULL';
      if ($cauto) {
        $defs[] = $driver === 'pgsql'
          ? qi($cname, $driver) . " SERIAL PRIMARY KEY"
          : qi($cname, $driver) . " INT AUTO_INCREMENT PRIMARY KEY";
      } else {
        $defs[] = trim(qi($cname, $driver) . " $ctype $cnull " . ($cpk ? 'PRIMARY KEY' : ''));
      }
    }
    try {
      $pdo_tmp = getConnection();
      $pdo_tmp->exec("CREATE TABLE " . qi($tname, $driver) . " (\n  " . implode(",\n  ", $defs) . "\n)");
      header('Location: ?action=dashboard&table=' . urlencode($tname) . '&created_table=1');
      exit;
    } catch (Exception $e) {
      $error = $e->getMessage();
    }
  }
}

// RUN SQL
if ($action === 'run_sql' && $_SERVER['REQUEST_METHOD'] === 'POST' && isset($_SESSION['db'])) {
  $sql_query    = trim($_POST['sql'] ?? '');
  $sql_result   = null;
  $sql_error    = null;
  $sql_affected = null;
  if ($sql_query) {
    try {
      $pdo_tmp = getConnection();
      $stmt = $pdo_tmp->query($sql_query);
      if ($stmt && $stmt->columnCount() > 0) $sql_result = $stmt->fetchAll();
      else $sql_affected = $stmt ? $stmt->rowCount() : 0;
    } catch (Exception $e) {
      $sql_error = $e->getMessage();
    }
  }
}

// ─────────────────────────────────────────────
// Data
// ─────────────────────────────────────────────
$pdo = getConnection();
$db  = $_SESSION['db'] ?? null;
if ($action !== 'login' && !$pdo) {
  header('Location: ?action=login');
  exit;
}

$databases   = [];
$tables      = [];
$activeTable = $_GET['table'] ?? null;

if ($pdo && $db) {
  try {
    $databases = getDatabases($pdo, $db['driver']);
  } catch (Exception $e) {
  }
  try {
    $tables = getTables($pdo, $db['driver']);
  } catch (Exception $e) {
  }
}
?>
<!DOCTYPE html>
<html lang="id" data-theme="dark">

<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>LunaBase<?= $db ? ' — ' . htmlspecialchars($db['dbname']) : '' ?></title>
  <style>
    :root {
      --bg: #0d0d10;
      --bg2: #13131a;
      --bg3: #1a1a24;
      --border: #2a2a38;
      --text: #e2e2f0;
      --text2: #8888aa;
      --text3: #404060;
      --accent: #7c6aff;
      --accent2: #a78bfa;
      --green: #34d399;
      --red: #f87171;
      --yellow: #fbbf24;
      --cyan: #22d3ee;
      --radius: 10px;
      --sidebar-w: 260px;
      --mono: 'JetBrains Mono', 'Fira Code', monospace;
      --sans: 'Inter', system-ui, sans-serif;
    }

    [data-theme="light"] {
      --bg: #f4f4f8;
      --bg2: #ffffff;
      --bg3: #ededf5;
      --border: #dddde8;
      --text: #1a1a2e;
      --text2: #55556a;
      --text3: #aaaacc;
    }

    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0
    }

    body {
      font-family: var(--sans);
      background: var(--bg);
      color: var(--text);
      min-height: 100vh;
      font-size: 14px;
      line-height: 1.6;
      -webkit-font-smoothing: antialiased
    }

    /* Login */
    .login-wrap {
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--bg);
      background-image: radial-gradient(ellipse at 20% 50%, #7c6aff18 0%, transparent 60%), radial-gradient(ellipse at 80% 20%, #22d3ee12 0%, transparent 50%)
    }

    .login-card {
      width: 420px;
      background: var(--bg2);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 40px;
      box-shadow: 0 24px 64px #0008
    }

    .login-logo {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 32px
    }

    .moon {
      width: 36px;
      height: 36px;
      background: linear-gradient(135deg, var(--accent), var(--cyan));
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 18px
    }

    .login-logo h1 {
      font-size: 22px;
      font-weight: 700;
      letter-spacing: -.5px
    }

    .login-logo span,
    .logo-span {
      color: var(--accent2)
    }

    .login-subtitle {
      color: var(--text2);
      font-size: 13px;
      margin-bottom: 28px
    }

    /* Form */
    .field {
      margin-bottom: 16px
    }

    .field label {
      display: block;
      font-size: 12px;
      font-weight: 500;
      color: var(--text2);
      margin-bottom: 6px;
      text-transform: uppercase;
      letter-spacing: .05em
    }

    .field input,
    .field select {
      width: 100%;
      background: var(--bg3);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      color: var(--text);
      padding: 10px 14px;
      font-size: 14px;
      font-family: var(--sans);
      outline: none;
      transition: border-color .2s
    }

    .field input:focus,
    .field select:focus {
      border-color: var(--accent)
    }

    .field select {
      cursor: pointer;
      appearance: none;
      background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%238888aa' stroke-width='2'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
      background-repeat: no-repeat;
      background-position: right 14px center;
      padding-right: 36px
    }

    .field-row {
      display: grid;
      grid-template-columns: 1fr 100px;
      gap: 10px
    }

    .btn {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 10px 18px;
      border-radius: var(--radius);
      border: none;
      cursor: pointer;
      font-size: 14px;
      font-weight: 500;
      font-family: var(--sans);
      transition: all .15s;
      text-decoration: none
    }

    .btn-primary {
      background: var(--accent);
      color: #fff;
      width: 100%;
      justify-content: center;
      padding: 12px;
      margin-top: 8px
    }

    .btn-primary:hover {
      background: #6b59ee
    }

    .btn-ghost {
      background: transparent;
      color: var(--text2);
      border: 1px solid var(--border)
    }

    .btn-ghost:hover {
      background: var(--bg3);
      color: var(--text)
    }

    .btn-sm {
      padding: 6px 12px;
      font-size: 12px
    }

    .error-box {
      background: #f8717118;
      border: 1px solid #f8717133;
      border-radius: var(--radius);
      padding: 10px 14px;
      color: var(--red);
      font-size: 13px;
      margin-bottom: 16px
    }

    .success-box {
      background: #34d39918;
      border: 1px solid #34d39933;
      border-radius: var(--radius);
      padding: 10px 14px;
      color: var(--green);
      font-size: 13px;
      margin-bottom: 16px
    }

    /* App */
    .app {
      display: flex;
      min-height: 100vh
    }

    /* Sidebar */
    .sidebar {
      width: var(--sidebar-w);
      background: var(--bg2);
      border-right: 1px solid var(--border);
      display: flex;
      flex-direction: column;
      position: fixed;
      top: 0;
      left: 0;
      bottom: 0;
      overflow-y: auto;
      z-index: 100
    }

    .sidebar-header {
      padding: 18px 16px 14px;
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      justify-content: space-between
    }

    .sidebar-logo {
      display: flex;
      align-items: center;
      gap: 8px;
      font-weight: 700;
      font-size: 15px
    }

    .sidebar-logo .moon {
      width: 28px;
      height: 28px;
      font-size: 13px
    }

    .driver-badge {
      font-size: 10px;
      font-weight: 600;
      padding: 2px 8px;
      border-radius: 20px;
      text-transform: uppercase;
      letter-spacing: .05em
    }

    .driver-pgsql {
      background: #22d3ee18;
      color: var(--cyan);
      border: 1px solid #22d3ee33
    }

    .driver-mysql {
      background: #f59e0b18;
      color: var(--yellow);
      border: 1px solid #f59e0b33
    }

    /* DB Switcher */
    .db-switcher {
      padding: 12px 16px;
      border-bottom: 1px solid var(--border)
    }

    .db-switcher label {
      font-size: 10px;
      font-weight: 600;
      color: var(--text2);
      text-transform: uppercase;
      letter-spacing: .08em;
      display: block;
      margin-bottom: 6px
    }

    .db-switcher select {
      width: 100%;
      background: var(--bg3);
      border: 1px solid var(--border);
      border-radius: 8px;
      color: var(--text);
      padding: 7px 10px;
      font-size: 13px;
      font-family: var(--sans);
      outline: none;
      cursor: pointer;
      appearance: none;
      background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 24 24' fill='none' stroke='%238888aa' stroke-width='2'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
      background-repeat: no-repeat;
      background-position: right 10px center;
      padding-right: 28px
    }

    .db-actions {
      display: flex;
      gap: 6px;
      margin-top: 8px
    }

    .db-btn {
      flex: 1;
      padding: 6px;
      border-radius: 6px;
      font-size: 11px;
      cursor: pointer;
      font-family: var(--sans);
      transition: all .15s;
      border: none
    }

    .db-btn-create {
      background: rgba(124, 106, 255, .1);
      border: 1px solid rgba(124, 106, 255, .2) !important;
      color: var(--accent2)
    }

    .db-btn-create:hover {
      background: rgba(124, 106, 255, .2)
    }

    .db-btn-drop {
      background: rgba(248, 113, 113, .08);
      border: 1px solid rgba(248, 113, 113, .15) !important;
      color: var(--red);
      flex: 0;
      padding: 6px 10px
    }

    .db-btn-drop:hover {
      background: rgba(248, 113, 113, .15)
    }

    /* Table list */
    .sidebar-section {
      padding: 12px 16px 6px
    }

    .sidebar-section-label {
      font-size: 10px;
      font-weight: 600;
      color: var(--text2);
      text-transform: uppercase;
      letter-spacing: .08em;
      margin-bottom: 6px;
      display: flex;
      align-items: center;
      justify-content: space-between
    }

    .table-list {
      list-style: none
    }

    .table-list li a {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 7px 10px;
      border-radius: 8px;
      color: var(--text2);
      text-decoration: none;
      font-size: 13px;
      transition: all .15s;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis
    }

    .table-list li a:hover {
      background: var(--bg3);
      color: var(--text)
    }

    .table-list li a.active {
      background: #7c6aff18;
      color: var(--accent2)
    }

    .tbl-icon {
      width: 16px;
      height: 16px;
      opacity: .5;
      flex-shrink: 0
    }

    .table-list li a.active .tbl-icon {
      opacity: 1
    }

    .sidebar-footer {
      margin-top: auto;
      padding: 12px 16px;
      border-top: 1px solid var(--border);
      display: flex;
      align-items: center;
      justify-content: space-between
    }

    .conn-info {
      font-size: 11px;
      color: var(--text2);
      line-height: 1.4
    }

    .conn-info strong {
      color: var(--text);
      font-size: 12px
    }

    /* Main */
    .main {
      margin-left: var(--sidebar-w);
      flex: 1;
      display: flex;
      flex-direction: column;
      min-height: 100vh
    }

    .topbar {
      height: 56px;
      border-bottom: 1px solid var(--border);
      padding: 0 24px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      background: var(--bg2);
      position: sticky;
      top: 0;
      z-index: 50
    }

    .topbar-title {
      font-size: 15px;
      font-weight: 600;
      display: flex;
      align-items: center;
      gap: 8px
    }

    .breadcrumb {
      color: var(--text2);
      font-weight: 400
    }

    .topbar-actions {
      display: flex;
      align-items: center;
      gap: 8px
    }

    .theme-toggle {
      width: 32px;
      height: 32px;
      border-radius: 8px;
      border: 1px solid var(--border);
      background: var(--bg3);
      color: var(--text2);
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all .15s
    }

    .theme-toggle:hover {
      color: var(--text);
      background: var(--border)
    }

    .content {
      padding: 24px
    }

    /* Stats */
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
      gap: 16px;
      margin-bottom: 28px
    }

    .stat-card {
      background: var(--bg2);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 20px
    }

    .stat-card .label {
      font-size: 11px;
      font-weight: 600;
      color: var(--text2);
      text-transform: uppercase;
      letter-spacing: .08em;
      margin-bottom: 8px
    }

    .stat-card .value {
      font-size: 28px;
      font-weight: 700;
      font-family: var(--mono);
      letter-spacing: -1px
    }

    .stat-card .sub {
      font-size: 12px;
      color: var(--text2);
      margin-top: 4px
    }

    /* Tables grid */
    .section-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 16px
    }

    .section-title {
      font-size: 13px;
      font-weight: 600;
      color: var(--text2);
      text-transform: uppercase;
      letter-spacing: .08em
    }

    .tables-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
      gap: 12px
    }

    .table-card {
      background: var(--bg2);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 16px 18px;
      text-decoration: none;
      color: var(--text);
      transition: all .15s;
      display: block
    }

    .table-card:hover {
      border-color: var(--accent);
      background: var(--bg3);
      transform: translateY(-1px);
      box-shadow: 0 4px 16px #7c6aff18
    }

    .table-name {
      font-size: 14px;
      font-weight: 600;
      font-family: var(--mono);
      margin-bottom: 8px;
      display: flex;
      align-items: center;
      gap: 8px
    }

    .table-name::before {
      content: '';
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: var(--accent);
      flex-shrink: 0
    }

    .table-meta {
      font-size: 12px;
      color: var(--text2);
      display: flex;
      align-items: center;
      gap: 12px
    }

    .row-count {
      font-family: var(--mono);
      color: var(--accent2);
      font-size: 13px;
      font-weight: 600
    }

    /* Search */
    .search-wrap {
      position: relative;
      margin-bottom: 20px
    }

    .search-wrap input {
      width: 100%;
      background: var(--bg2);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      color: var(--text);
      padding: 10px 14px 10px 38px;
      font-size: 14px;
      outline: none;
      transition: border-color .2s
    }

    .search-wrap input:focus {
      border-color: var(--accent)
    }

    .search-icon {
      position: absolute;
      left: 12px;
      top: 50%;
      transform: translateY(-50%);
      color: var(--text2);
      pointer-events: none
    }

    /* Empty */
    .empty {
      text-align: center;
      padding: 60px 24px;
      color: var(--text2)
    }

    .empty .icon {
      font-size: 40px;
      margin-bottom: 12px
    }

    .empty h3 {
      font-size: 16px;
      color: var(--text);
      margin-bottom: 6px
    }

    /* SQL Editor */
    .sql-editor {
      width: 100%;
      background: var(--bg3);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      color: var(--text);
      padding: 16px;
      font-family: var(--mono);
      font-size: 13px;
      line-height: 1.6;
      outline: none;
      resize: vertical;
      min-height: 160px;
      transition: border-color .2s
    }

    .sql-editor:focus {
      border-color: var(--accent)
    }

    .result-wrap {
      margin-top: 24px;
      overflow-x: auto;
      background: var(--bg2);
      border: 1px solid var(--border);
      border-radius: var(--radius)
    }

    .result-header {
      padding: 10px 16px;
      border-bottom: 1px solid var(--border);
      font-size: 12px;
      color: var(--text2)
    }

    .result-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
      font-family: var(--mono)
    }

    .result-table th {
      padding: 10px 16px;
      text-align: left;
      color: var(--text2);
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: .05em;
      white-space: nowrap;
      border-bottom: 1px solid var(--border)
    }

    .result-table td {
      padding: 9px 16px;
      color: var(--text);
      border-bottom: 1px solid var(--border);
      max-width: 300px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap
    }

    .result-table tr:nth-child(even) td {
      background: var(--bg3)
    }

    /* Create table */
    .col-row {
      display: grid;
      grid-template-columns: 1fr 160px auto auto auto auto;
      gap: 8px;
      align-items: center;
      background: var(--bg3);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 10px 12px;
      margin-bottom: 8px
    }

    .col-row input,
    .col-row select {
      background: var(--bg2);
      border: 1px solid var(--border);
      border-radius: 6px;
      color: var(--text);
      padding: 7px 10px;
      font-size: 13px;
      font-family: var(--mono);
      outline: none;
      width: 100%
    }

    .col-row input:focus,
    .col-row select:focus {
      border-color: var(--accent)
    }

    .col-checkbox {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 3px;
      font-size: 10px;
      color: var(--text2);
      text-transform: uppercase;
      letter-spacing: .05em
    }

    .col-checkbox input[type=checkbox] {
      width: 16px;
      height: 16px;
      accent-color: var(--accent);
      cursor: pointer
    }

    .btn-danger {
      background: #f8717118;
      border: 1px solid #f8717133;
      color: var(--red);
      padding: 6px 10px;
      border-radius: 6px;
      cursor: pointer;
      font-size: 13px;
      transition: all .15s
    }

    .btn-danger:hover {
      background: #f8717130
    }

    /* Modal */
    .modal-overlay {
      display: none;
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, .65);
      z-index: 200;
      align-items: center;
      justify-content: center
    }

    .modal-box {
      background: var(--bg2);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 32px;
      width: 400px;
      box-shadow: 0 24px 64px #0008
    }

    .modal-title {
      font-size: 16px;
      font-weight: 700;
      margin-bottom: 6px
    }

    .modal-sub {
      font-size: 13px;
      color: var(--text2);
      margin-bottom: 20px
    }

    .modal-input {
      width: 100%;
      background: var(--bg3);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      color: var(--text);
      padding: 10px 14px;
      font-size: 14px;
      font-family: var(--mono);
      outline: none;
      margin-bottom: 20px
    }

    .modal-input:focus {
      border-color: var(--accent)
    }

    .modal-actions {
      display: flex;
      gap: 10px
    }

    /* Table view */
    .data-table-wrap {
      overflow-x: auto;
      background: var(--bg2);
      border: 1px solid var(--border);
      border-radius: var(--radius)
    }

    .data-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
      font-family: var(--mono)
    }

    .data-table th {
      padding: 10px 16px;
      text-align: left;
      color: var(--text2);
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: .05em;
      white-space: nowrap;
      border-bottom: 1px solid var(--border)
    }

    .data-table td {
      padding: 9px 16px;
      color: var(--text);
      border-bottom: 1px solid var(--border);
      max-width: 280px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap
    }

    .data-table tr:nth-child(even) td {
      background: var(--bg3)
    }

    .null-val {
      color: var(--text2);
      font-style: italic
    }

    .bool-true {
      color: var(--green)
    }

    .bool-false {
      color: var(--red)
    }

    /* Scrollbar */
    ::-webkit-scrollbar {
      width: 5px;
      height: 5px
    }

    ::-webkit-scrollbar-track {
      background: transparent
    }

    ::-webkit-scrollbar-thumb {
      background: var(--border);
      border-radius: 3px
    }

    ::-webkit-scrollbar-thumb:hover {
      background: var(--text2)
    }
  </style>
</head>

<body>

  <?php if ($action === 'login'): ?>
    <div class="login-wrap">
      <div class="login-card">
        <div class="login-logo">
          <div class="moon">🌙</div>
          <h1>Luna<span>Base</span></h1>
        </div>
        <p class="login-subtitle">Koneksi ke database lokal kamu</p>
        <?php if ($error): ?><div class="error-box">⚠ <?= htmlspecialchars($error) ?></div><?php endif; ?>
        <form method="POST" action="?action=login">
          <div class="field">
            <label>Driver</label>
            <select name="driver" id="drv" onchange="updatePort(this.value)">
              <option value="pgsql">PostgreSQL</option>
              <option value="mysql">MySQL / MariaDB</option>
            </select>
          </div>
          <div class="field field-row">
            <div><label>Host</label><input type="text" name="host" value="localhost"></div>
            <div><label>Port</label><input type="text" name="port" id="port" value="5432"></div>
          </div>
          <div class="field"><label>Username</label><input type="text" name="user" value="<?= htmlspecialchars($_SERVER['USER'] ?? '') ?>"></div>
          <div class="field"><label>Password</label><input type="password" name="password"></div>
          <div class="field">
            <label>Database <span style="color:var(--text3);font-size:10px;text-transform:none;letter-spacing:0">(kosongkan = default, atau ketik nama baru)</span></label>
            <input type="text" name="dbname" id="dbname" placeholder="postgres">
          </div>
          <button type="submit" class="btn btn-primary">Masuk →</button>
        </form>
      </div>
    </div>
    <script>
      function updatePort(d) {
        document.getElementById('port').value = d === 'pgsql' ? '5432' : '3306';
        document.getElementById('dbname').placeholder = d === 'pgsql' ? 'postgres' : '';
      }
    </script>

  <?php else: ?>
    <div class="app">
      <!-- Sidebar -->
      <aside class="sidebar">
        <div class="sidebar-header">
          <div class="sidebar-logo">
            <div class="moon">🌙</div>
            Luna<span class="logo-span">Base</span>
          </div>
          <span class="driver-badge driver-<?= $db['driver'] ?>">
            <?= $db['driver'] === 'pgsql' ? 'PG' : 'MY' ?>
          </span>
        </div>

        <!-- DB Switcher -->
        <div class="db-switcher">
          <label>Database</label>
          <form method="POST" action="?action=switch_db" id="db-form">
            <select name="dbname" onchange="document.getElementById('db-form').submit()">
              <?php foreach ($databases as $dn): ?>
                <option value="<?= htmlspecialchars($dn) ?>" <?= $dn === $db['dbname'] ? 'selected' : '' ?>>
                  <?= htmlspecialchars($dn) ?>
                </option>
              <?php endforeach; ?>
            </select>
          </form>
          <div class="db-actions">
            <button class="db-btn db-btn-create" onclick="showModal('modal-create')">+ Buat Database</button>
            <button class="db-btn db-btn-drop" onclick="showDropModal('<?= htmlspecialchars($db['dbname']) ?>')" title="Hapus database ini">✕</button>
          </div>
        </div>

        <!-- Tables -->
        <div class="sidebar-section">
          <div class="sidebar-section-label">
            <span>Tables</span>
            <span style="font-family:var(--mono);color:var(--accent2)"><?= count($tables) ?></span>
          </div>
          <ul class="table-list">
            <?php foreach ($tables as $tbl): ?>
              <li>
                <a href="?action=dashboard&table=<?= urlencode($tbl) ?>" class="<?= $activeTable === $tbl ? 'active' : '' ?>">
                  <svg class="tbl-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <rect x="3" y="3" width="18" height="18" rx="2" />
                    <path d="M3 9h18M3 15h18M9 3v18" />
                  </svg>
                  <?= htmlspecialchars($tbl) ?>
                </a>
              </li>
            <?php endforeach; ?>
            <?php if (empty($tables)): ?>
              <li style="padding:8px 10px;color:var(--text2);font-size:12px">Belum ada table</li>
            <?php endif; ?>
          </ul>
        </div>

        <div class="sidebar-footer">
          <div class="conn-info">
            <strong><?= htmlspecialchars($db['user']) ?></strong><br>
            <?= htmlspecialchars($db['host']) ?>:<?= htmlspecialchars($db['port']) ?>
          </div>
          <a href="?action=logout" class="btn btn-ghost btn-sm">Keluar</a>
        </div>
      </aside>

      <!-- Main -->
      <main class="main">
        <div class="topbar">
          <div class="topbar-title">
            <?php if ($activeTable): ?>
              <span class="breadcrumb"><?= htmlspecialchars($db['dbname']) ?> /</span>
              <?= htmlspecialchars($activeTable) ?>
            <?php elseif (in_array($action, ['create_table', 'sql_editor'])): ?>
              <span class="breadcrumb"><?= htmlspecialchars($db['dbname']) ?> /</span>
              <?= $action === 'create_table' ? 'Buat Table' : 'SQL Editor' ?>
            <?php else: ?>
              <?= htmlspecialchars($db['dbname']) ?>
            <?php endif; ?>
          </div>
          <div class="topbar-actions">
            <a href="?action=sql_editor" class="btn btn-ghost btn-sm">⌨ SQL Editor</a>
            <a href="?action=create_table" class="btn btn-ghost btn-sm">+ Buat Table</a>
            <button class="theme-toggle" onclick="toggleTheme()" title="Toggle theme">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="5" />
                <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
              </svg>
            </button>
          </div>
        </div>

        <div class="content">
          <!-- Notifikasi -->
          <?php if (isset($_GET['created_db'])): ?>
            <div class="success-box">✓ Database <strong><?= htmlspecialchars($_GET['created_db']) ?></strong> berhasil dibuat dan aktif.</div>
          <?php endif; ?>
          <?php if (isset($_GET['dropped_db'])): ?>
            <div class="error-box">Database <strong><?= htmlspecialchars($_GET['dropped_db']) ?></strong> berhasil dihapus.</div>
          <?php endif; ?>
          <?php if (isset($_GET['created_table'])): ?>
            <div class="success-box">✓ Table berhasil dibuat.</div>
          <?php endif; ?>
          <?php if ($error && !in_array($action, ['login'])): ?>
            <div class="error-box">⚠ <?= htmlspecialchars($error) ?></div>
          <?php endif; ?>

          <?php if ($action === 'create_table'): ?>
            <!-- CREATE TABLE -->
            <div style="max-width:860px">
              <div class="section-header" style="margin-bottom:20px">
                <span class="section-title">Buat Table Baru</span>
                <a href="?action=dashboard" class="btn btn-ghost btn-sm">← Kembali</a>
              </div>
              <form method="POST" action="?action=create_table">
                <div class="field" style="margin-bottom:20px">
                  <label>Nama Table</label>
                  <input type="text" name="table_name" placeholder="users" style="width:100%;background:var(--bg3);border:1px solid var(--border);border-radius:var(--radius);color:var(--text);padding:10px 14px;font-size:14px;font-family:var(--mono);outline:none">
                </div>
                <div class="section-title" style="margin-bottom:12px">Kolom</div>
                <div style="display:grid;grid-template-columns:1fr 160px 50px 50px 50px 36px;gap:8px;padding:0 12px 8px;font-size:10px;font-weight:600;color:var(--text2);text-transform:uppercase;letter-spacing:.05em">
                  <span>Nama Kolom</span><span>Tipe Data</span><span>PK</span><span>NULL</span><span>AUTO</span><span></span>
                </div>
                <div id="col-list">
                  <div class="col-row">
                    <input type="text" name="columns[0][name]" placeholder="id">
                    <select name="columns[0][type]">
                      <option>SERIAL</option>
                      <option>INT</option>
                      <option>BIGINT</option>
                      <option>VARCHAR(255)</option>
                      <option>TEXT</option>
                      <option>BOOLEAN</option>
                      <option>TIMESTAMP</option>
                      <option>DATE</option>
                      <option>NUMERIC</option>
                      <option>JSONB</option>
                      <option>UUID</option>
                    </select>
                    <div class="col-checkbox"><input type="checkbox" name="columns[0][primary]"></div>
                    <div class="col-checkbox"><input type="checkbox" name="columns[0][nullable]"></div>
                    <div class="col-checkbox"><input type="checkbox" name="columns[0][auto]" checked></div>
                    <button type="button" class="btn-danger" onclick="removeCol(this)">✕</button>
                  </div>
                </div>
                <button type="button" class="btn btn-ghost btn-sm" onclick="addCol()" style="margin:12px 0 28px">+ Tambah Kolom</button>
                <br>
                <button type="submit" class="btn btn-primary" style="width:auto;padding:10px 28px">Buat Table →</button>
              </form>
            </div>

          <?php elseif ($action === 'sql_editor'): ?>
            <!-- SQL EDITOR -->
            <div>
              <div class="section-header" style="margin-bottom:20px">
                <span class="section-title">SQL Editor</span>
                <span style="font-size:12px;color:var(--text2)">Ctrl+Enter untuk jalankan</span>
              </div>
              <?php if (isset($sql_error)): ?><div class="error-box">⚠ <?= htmlspecialchars($sql_error) ?></div><?php endif; ?>
              <?php if (isset($sql_affected)): ?><div class="success-box">✓ Query berhasil — <?= $sql_affected ?> baris terpengaruh</div><?php endif; ?>
              <form method="POST" action="?action=run_sql" id="sql-form">
                <textarea class="sql-editor" name="sql" placeholder="SELECT * FROM users LIMIT 10;&#10;-- Ctrl+Enter untuk jalankan"><?= htmlspecialchars($_POST['sql'] ?? '') ?></textarea>
                <div style="display:flex;justify-content:flex-end;margin-top:10px;gap:8px">
                  <button type="button" onclick="document.querySelector('.sql-editor').value=''" class="btn btn-ghost btn-sm">Hapus</button>
                  <button type="submit" class="btn btn-primary" style="width:auto;padding:10px 24px">▶ Jalankan</button>
                </div>
              </form>
              <?php if (isset($sql_result) && $sql_result !== null): ?>
                <div class="result-wrap">
                  <div class="result-header"><?= count($sql_result) ?> baris dikembalikan</div>
                  <?php if (!empty($sql_result)): ?>
                    <table class="result-table">
                      <thead>
                        <tr><?php foreach (array_keys($sql_result[0]) as $col): ?><th><?= htmlspecialchars($col) ?></th><?php endforeach; ?></tr>
                      </thead>
                      <tbody>
                        <?php foreach ($sql_result as $row): ?>
                          <tr><?php foreach ($row as $val): ?>
                              <td><?= is_null($val) ? '<span class="null-val">NULL</span>' : htmlspecialchars((string)$val) ?></td>
                            <?php endforeach; ?>
                          </tr>
                        <?php endforeach; ?>
                      </tbody>
                    </table>
                  <?php endif; ?>
                </div>
              <?php endif; ?>
            </div>

          <?php elseif (!$activeTable): ?>
            <!-- DASHBOARD -->
            <div class="stats-grid">
              <div class="stat-card">
                <div class="label">Tables</div>
                <div class="value"><?= count($tables) ?></div>
                <div class="sub">di <?= htmlspecialchars($db['dbname']) ?></div>
              </div>
              <div class="stat-card">
                <div class="label">Databases</div>
                <div class="value"><?= count($databases) ?></div>
                <div class="sub">tersedia</div>
              </div>
              <div class="stat-card">
                <div class="label">Driver</div>
                <div class="value" style="font-size:16px;color:var(--accent2);letter-spacing:0">
                  <?= $db['driver'] === 'pgsql' ? 'PostgreSQL' : 'MySQL' ?>
                </div>
                <div class="sub"><?= htmlspecialchars($db['host']) ?>:<?= htmlspecialchars($db['port']) ?></div>
              </div>
            </div>

            <?php if (!empty($tables)): ?>
              <div class="section-header">
                <span class="section-title">Tables</span>
              </div>
              <div class="search-wrap">
                <span class="search-icon">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="11" cy="11" r="8" />
                    <path d="M21 21l-4.35-4.35" />
                  </svg>
                </span>
                <input type="text" placeholder="Cari table..." oninput="filterTables(this.value)">
              </div>
              <div class="tables-grid" id="tables-grid">
                <?php foreach ($tables as $tbl): ?>
                  <?php $cnt = getRowCount($pdo, $tbl, $db['driver']); ?>
                  <a href="?action=dashboard&table=<?= urlencode($tbl) ?>" class="table-card" data-name="<?= htmlspecialchars($tbl) ?>">
                    <div class="table-name"><?= htmlspecialchars($tbl) ?></div>
                    <div class="table-meta">
                      <span class="row-count"><?= number_format($cnt) ?></span>
                      <span>rows</span>
                    </div>
                  </a>
                <?php endforeach; ?>
              </div>
            <?php else: ?>
              <div class="empty">
                <div class="icon">📭</div>
                <h3>Database kosong</h3>
                <p>Belum ada table. <a href="?action=create_table" style="color:var(--accent2)">Buat table baru →</a></p>
              </div>
            <?php endif; ?>

          <?php else: ?>
            <!-- TABLE VIEW -->
            <?php
            $page    = max(1, (int)($_GET['page'] ?? 1));
            $perPage = 25;
            $offset  = ($page - 1) * $perPage;
            $total   = 0;
            $totalPages = 1;
            $rows    = [];
            $columns = [];
            try {
              $qt = qi($activeTable, $db['driver']);
              $total = $pdo->query("SELECT COUNT(*) FROM $qt")->fetchColumn();
              $totalPages = max(1, ceil($total / $perPage));
              $rows = $pdo->query("SELECT * FROM $qt LIMIT $perPage OFFSET $offset")->fetchAll();
              $columns = $rows ? array_keys($rows[0]) : [];
            } catch (Exception $e) {
              $error = $e->getMessage();
            }
            ?>
            <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:16px">
              <div style="font-size:12px;color:var(--text2)">
                <?= number_format($total) ?> rows · Page <?= $page ?> / <?= $totalPages ?>
              </div>
              <div style="display:flex;gap:8px">
                <?php if ($page > 1): ?>
                  <a href="?action=dashboard&table=<?= urlencode($activeTable) ?>&page=<?= $page - 1 ?>" class="btn btn-ghost btn-sm">← Prev</a>
                <?php endif; ?>
                <?php if ($page < $totalPages): ?>
                  <a href="?action=dashboard&table=<?= urlencode($activeTable) ?>&page=<?= $page + 1 ?>" class="btn btn-ghost btn-sm">Next →</a>
                <?php endif; ?>
              </div>
            </div>
            <?php if (!empty($rows)): ?>
              <div class="data-table-wrap">
                <table class="data-table">
                  <thead>
                    <tr><?php foreach ($columns as $col): ?><th><?= htmlspecialchars($col) ?></th><?php endforeach; ?></tr>
                  </thead>
                  <tbody>
                    <?php foreach ($rows as $row): ?>
                      <tr>
                        <?php foreach ($row as $val): ?>
                          <td>
                            <?php if (is_null($val)): ?><span class="null-val">NULL</span>
                            <?php elseif ($val === true || $val === 't'): ?><span class="bool-true">true</span>
                            <?php elseif ($val === false || $val === 'f'): ?><span class="bool-false">false</span>
                              <?php else: ?><?= htmlspecialchars((string)$val) ?>
                            <?php endif; ?>
                          </td>
                        <?php endforeach; ?>
                      </tr>
                    <?php endforeach; ?>
                  </tbody>
                </table>
              </div>
            <?php else: ?>
              <div class="empty">
                <div class="icon">📭</div>
                <h3>Table kosong</h3>
                <p>Belum ada data.</p>
              </div>
            <?php endif; ?>
          <?php endif; ?>
        </div>
      </main>
    </div>

    <!-- Modal: Buat Database -->
    <div class="modal-overlay" id="modal-create">
      <div class="modal-box">
        <div class="modal-title">Buat Database Baru</div>
        <div class="modal-sub">Database langsung aktif setelah dibuat.</div>
        <form method="POST" action="?action=create_database">
          <input type="text" name="dbname" class="modal-input" placeholder="nama_database" autofocus>
          <div class="modal-actions">
            <button type="submit" class="btn btn-primary" style="flex:1;justify-content:center">Buat →</button>
            <button type="button" onclick="hideModals()" class="btn btn-ghost" style="flex:1;justify-content:center">Batal</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Modal: Hapus Database -->
    <div class="modal-overlay" id="modal-drop">
      <div class="modal-box">
        <div class="modal-title" style="color:var(--red)">Hapus Database</div>
        <div class="modal-sub">
          Database <strong id="drop-name" style="font-family:var(--mono);color:var(--text)"></strong> akan dihapus permanen beserta semua datanya.
        </div>
        <form method="POST" action="?action=drop_database">
          <input type="hidden" name="dbname" id="drop-input">
          <div class="modal-actions">
            <button type="submit" style="flex:1;padding:10px;background:rgba(248,113,113,.15);border:1px solid rgba(248,113,113,.3);border-radius:8px;color:var(--red);font-size:13px;font-weight:600;cursor:pointer;font-family:var(--sans)">Hapus Permanen</button>
            <button type="button" onclick="hideModals()" class="btn btn-ghost" style="flex:1;justify-content:center">Batal</button>
          </div>
        </form>
      </div>
    </div>
  <?php endif; ?>

  <script>
    // Theme
    function toggleTheme() {
      const h = document.documentElement,
        n = h.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
      h.setAttribute('data-theme', n);
      localStorage.setItem('lb-theme', n);
    }
    const t = localStorage.getItem('lb-theme');
    if (t) document.documentElement.setAttribute('data-theme', t);

    // Filter tables
    function filterTables(q) {
      document.querySelectorAll('.table-card').forEach(c => {
        c.style.display = c.dataset.name.toLowerCase().includes(q.toLowerCase()) ? '' : 'none';
      });
    }

    // Modals
    function showModal(id) {
      document.getElementById(id).style.display = 'flex'
    }

    function hideModals() {
      document.querySelectorAll('.modal-overlay').forEach(m => m.style.display = 'none');
    }

    function showDropModal(name) {
      const safe = ['postgres', 'mysql', 'information_schema', 'performance_schema', 'sys'];
      if (safe.includes(name)) {
        alert('Database default tidak bisa dihapus.');
        return;
      }
      document.getElementById('drop-name').textContent = name;
      document.getElementById('drop-input').value = name;
      showModal('modal-drop');
    }
    document.addEventListener('click', function(e) {
      document.querySelectorAll('.modal-overlay').forEach(m => {
        if (e.target === m) hideModals();
      });
    });

    // Create table
    const TYPES = ['SERIAL', 'INT', 'BIGINT', 'VARCHAR(255)', 'TEXT', 'BOOLEAN', 'TIMESTAMP', 'DATE', 'NUMERIC', 'JSONB', 'UUID', 'FLOAT'];
    let ci = 1;

    function addCol() {
      const i = ci++;
      const d = document.createElement('div');
      d.className = 'col-row';
      d.innerHTML = `
        <input type="text" name="columns[${i}][name]" placeholder="kolom_${i}">
        <select name="columns[${i}][type]">${TYPES.map(t=>`<option>${t}</option>`).join('')}</select>
        <div class="col-checkbox"><input type="checkbox" name="columns[${i}][primary]"></div>
        <div class="col-checkbox"><input type="checkbox" name="columns[${i}][nullable]"></div>
        <div class="col-checkbox"><input type="checkbox" name="columns[${i}][auto]"></div>
        <button type="button" class="btn-danger" onclick="removeCol(this)">✕</button>`;
      document.getElementById('col-list').appendChild(d);
    }

    function removeCol(btn) {
      if (document.querySelectorAll('.col-row').length <= 1) return;
      btn.closest('.col-row').remove();
    }

    // SQL Editor: Ctrl+Enter
    document.addEventListener('keydown', function(e) {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        const f = document.querySelector('.sql-editor')?.closest('form');
        if (f) f.submit();
      }
    });
  </script>
</body>

</html>
