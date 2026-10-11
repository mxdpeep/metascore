<?php
/**
 * Recursively traverse folder sizes
 * php version 8.2
 *
 * @category Helper
 * @package  None
 * @author   Fred Brooker <git@gscloud.cz>
 * @license  MIT https://gscloud.cz/LICENSE
 * @link     None
 */

mb_internal_encoding('UTF-8');
require __DIR__ . '/vendor/autoload.php';
require_once __DIR__ . '/config.php';

$cli = new \League\CLImate\CLImate;

$result = exec("which dust", $output, $return_var);
if ($return_var !== 0) {
    $cli->error("ERROR: dust command not found!");
    exit(1);
}

// blocked filenames
$blocked = [];

// allowed paths
$allowed = [
    'GAME',
    'GAME GOG',
];

$t = $t2 = 0;
$cache = $cache2 = [];

/**
 * Checks if a haystack string starts with any of the needles
 *
 * @param string        $haystack string to search in
 * @param array<string> $needles  array of strings to search for
 *
 * @return bool true if the haystack starts with any of the needles
 */
function startsWithAny(String $haystack, Array $needles)
{
    foreach ($needles as $needle) {
        if (strpos($haystack, $needle) === 0) {
            return true;
        }
    }
    return false;
}

/**
 * Get the size of a directory in rough gigabytes
 *
 * @param string $directory path to the directory
 *
 * @return float size of the directory in gigabytes or 0
 */
function getDirSize(String $directory)
{
    $escaped = escapeshellarg($directory);
    $command = "dust -j -o b $escaped 2>/dev/null"; 
    //$command = "dust -j -o b $escaped";
    $output = shell_exec($command);
    $size_b = 0;
    if ($output) {
        $json = json_decode($output, true);
        if (is_array($json) && isset($json['size'])) {
            $size_b = (int) trim($json['size'], 'B');
        }
    }
    $size_k = $size_b / 1024;
    $size_m = $size_k / 1024;
    $size_g = $size_m / 1024;
    return ceil($size_g);
}

// read items from cache
if (file_exists(CACHE) && is_readable(CACHE)) {
    $cache = json_decode(@file_get_contents(CACHE) ?: '', true);
    if (is_array($cache)) {
        $t = count($cache);
    } else {
        $cache = [];
    }
}
$cli->info("Cached items: $t");

// read sizes from cache
if (file_exists(CACHE2) && is_readable(CACHE2)) {
    $cache2 = json_decode(@file_get_contents(CACHE2) ?: '', true);
    if (is_array($cache2)) {
        $t2 = count($cache2);
    } else {
        $cache2 = [];
    }
}
$cli->info("Cached sizes: $t2");

$s = 0;
$count = 0;
if ($t = count($cache)) {
    $cli->out('<bold>Processing folder sizes ...</bold>');
    // @phpstan-ignore-next-line
    $progress = $cli->progress()->total($t);
    foreach ($cache as $h => $v) {
        $path = $v['path'];
        $name = $v['name'];
        $s++;
        $progress->current($s, "{$s}. $name");
        usleep(100);
        if (is_numeric($name)) {
            continue;
        }
        if (in_array($name, $blocked)) {
            continue;
        }
        if (startsWithAny($path, $allowed)) {
            if (isset($v['size'])) {
                $cache2[$h]['size'] = $v['size'];
                continue;
            }
            $f = FS . "{$path}{$name}";
            if (is_dir($f) && is_readable($f)) {
                $size = $cache2[$h]['size'] ?? getDirSize($f);
                $cache[$h]['size'] = $size;
                $cache2[$h]['size'] = $size;
                $count++;
            }
        }
    }
    $progress->current($t, " ");
}

if ($count) {
    $cli->info("Processed: <bold>$count</bold>");
}

// export items cache
if (is_array($cache)) {
    if (file_put_contents(CACHE, json_encode($cache, JSON_PRETTY_PRINT), LOCK_EX)) { // phpcs:ignore
        copy(CACHE, FS . CACHE);
        $cli->info("Items cache saved.");
    }
}

// export sizes cache
if (is_array($cache2)) {
    if (file_put_contents(CACHE2, json_encode($cache2, JSON_PRETTY_PRINT), LOCK_EX)) { // phpcs:ignore
        copy(CACHE2, FS . CACHE2);
        $cli->info("Sizes cache saved.");
    }
}

exit(0);
