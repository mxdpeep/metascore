<?php
/**
 * Recursively fix folder names with ratings
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

// blocked filenames
$blocked = [
    'GAME',
    'GAME GOG',
    'GAME NODRM',
];

// transliteration rules
$trans = [
    'á' => 'a',
    'à' => 'a',
    'á' => 'a',
    'ä' => 'a',
    'č' => 'c',
    'ć' => 'c',
    'č' => 'c',
    'ď' => 'd',
    'é' => 'e',
    'ě' => 'e',
    'è' => 'e',
    'é' => 'e',
    'ë' => 'e',
    'ě' => 'e',
    'í' => 'i',
    'í' => 'i',
    'ĺ' => 'l',
    'ľ' => 'l',
    'ḿ' => 'm',
    'ń' => 'n',
    'ň' => 'n',
    'ó' => 'o',
    'ö' => 'o',
    'ø' => 'o',
    'ř' => 'r',
    'ŕ' => 'r',
    'ř' => 'r',
    'ś' => 's',
    'š' => 's',
    'š' => 's',
    'ť' => 't',
    'ú' => 'u',
    'ú' => 'u',
    'ü' => 'u',
    'ů' => 'u',
    'ý' => 'y',
    'ý' => 'y',
    'ź' => 'z',
    'ž' => 'z',
    'ž' => 'z',
    'Á' => 'a',
    'À' => 'a',
    'Á' => 'a',
    'Ä' => 'a',
    'Č' => 'c',
    'Ć' => 'c',
    'Č' => 'c',
    'Ď' => 'd',
    'É' => 'e',
    'Ě' => 'e',
    'È' => 'e',
    'É' => 'e',
    'Ë' => 'e',
    'Ě' => 'e',
    'Í' => 'i',
    'Í' => 'i',
    'Ĺ' => 'l',
    'Ľ' => 'l',
    'Ḿ' => 'M',
    'Ň' => 'n',
    'Ń' => 'n',
    'Ó' => 'o',
    'Ö' => 'o',
    'Ø' => 'o',
    'Ř' => 'r',
    'Ŕ' => 'r',
    'Ř' => 'r',
    'Ś' => 's',
    'Š' => 's',
    'Š' => 's',
    'Ť' => 't',
    'Ú' => 'u',
    'Ú' => 'u',
    'Ü' => 'u',
    'Ů' => 'u',
    'Ý' => 'y',
    'Ý' => 'y',
    'Ź' => 'z',
    'Ž' => 'z',
    'Ž' => 'z',
];

$na = $tbd = $c = $t = $count = 0;
$cache = $files = $games = $renames = $scores = [];

// read GO subcode CSV database
if ($file = @fopen(SCORES, 'r')) {
    while (($row = fgetcsv($file, 0, ';')) !== false) {
        $name = $row[0];
        $score = $row[1];
        if ($score === 'TBD') {
            $tbd++;
            continue;
        }
        if ($score === 'N/A') {
            $na++;
            continue;
        }
        if (!is_numeric($score)) {
            continue;
        }
        $tr = str_replace(array_keys($trans), $trans, $name);
        $tr = strtolower($tr);
        $hash = hash('sha1', $tr);
        $files[$hash] = $name;
        $games[$hash] = $tr;
        $scores[$hash] = $score;
    }
}
$cli->info("N/A: {$na} items");
$cli->info("TBD: {$tbd} items");
$cli->info("Processing: " . count($games) . " items");

// read FS from cache
if (file_exists(CACHE) && is_readable(CACHE)) {
    $cache = json_decode(@file_get_contents(CACHE) ?: '', true);
    if (is_array($cache)) {
        $t = count($cache);
    } else {
        $cache = [];
    }
}

// traverse filesystem if there is no cache
if ($t === 0 && file_exists(FS) && is_dir(FS)) {
    // @phpstan-ignore-next-line
    $progress = $cli->progress()->total(100);
    
    try {
        // iterate through all folders with specific depth
        foreach ($DF??[] as $pth => $depth) {
            // recursive iterator
            $i = new RecursiveIteratorIterator(
                new RecursiveDirectoryIterator(FS . $pth, SKDS), SF
            );
            $i->setMaxDepth($depth);
            foreach ($i as $item) {
                $p = $pth . $i->getSubPath() . '/';
                $p = str_replace('//', '/', $p);
                $pn = $i->getFileName();
                if ($i->isDir() && strlen($pn) > 1) {
                    if (in_array($pn, $blocked)) {
                        continue;
                    }
                    $c++;
                    if ($c > 100) {
                        $c = 1;
                    }
                    $t++;
                    $progress->current($c, "{$t}. {$p} - {$pn}");
                    $tr = str_replace(array_keys($trans), $trans, $pn);
                    $tr = strtolower($tr);
                    $h = hash('sha1', $tr);
                    $cache[$h] = [
                        "path" => $p,
                        "name" => $pn,
                    ];
                    if (in_array($tr, $games)) {                
                        if (!isset($scores[$h])) {
                            continue;
                        }
                        $nn = "$pn [{$scores[$h]}%]";
                        $renames[] = [
                            "from" => FS . "{$p}{$pn}",
                            "to" => FS . "{$p}{$nn}",
                        ];
                        $cache[$h] = [
                            "path" => $p,
                            "name" => $nn,
                        ];
                        $count++;
                    }
                }
            }
        }
    } catch(Throwable $e) {
        dump($e);
        exit(1);
    }
    $progress->current(100, " ");
} else {
    $cli->info("Using FS cache.");
    $c = 0;
    // @phpstan-ignore-next-line
    $progress = $cli->progress()->total(count($cache));
    foreach ($cache as $h => $v) {
        $p = $v['path'];
        $pn = $v['name'];
        $tr = str_replace(array_keys($trans), $trans, $pn);
        $tr = strtolower($tr);
        $c++;
        $progress->current($c, "{$t}. {$p} - {$pn}");
        usleep(100);
        if (in_array($tr, $games)) {                
            if (!isset($scores[$h])) {
                continue;
            }
            if (!$scores[$h]) {
                continue;
            }
            $nn = "$pn [{$scores[$h]}%]";
            $renames[] = [
                "from" => FS . "{$p}{$pn}",
                "to" => FS . "{$p}{$nn}",
            ];
            $cache[$h] = [
                "path" => $p,
                "name" => $nn,
            ];
            $count++;
        }
    }
    $progress->current($c, " ");
}

$cli->info("Total items: {$t}\nRenames: {$count}");
$c = 0;
// @phpstan-ignore-next-line
$progress = $cli->progress()->total(count($renames));
foreach ($renames as $r) {
    if (rename($r['from'], $r['to'])) {
        $c++;
        $progress->current($c, $r['to']);
    } else {
        $cli->error("failed: {$r['from']} > {$r['to']}");
    }
}
if ($c) {
    $progress->current($c, "");
}

if (is_array($cache)) {
    $names = array_column($cache, 'name');
    sort($names);
    $names = array_unique($names);
    $out = implode("\n", $names);
    if (!file_put_contents(DB, $out, LOCK_EX)) {
        $cli->error("Nepodařilo se uložit soubor DB: " . DB);
        exit(1);
    }
    if (!file_put_contents(CACHE, json_encode($cache, JSON_PRETTY_PRINT), LOCK_EX)) {
        $cli->error("Nepodařilo se uložit soubor CACHE: " . CACHE);
        exit(1);
    }
    if (@copy(CACHE, FS . CACHE) && @copy(DB, FS . DB) && @copy(SCORES, FS . basename(SCORES))) { // phpcs:ignore
        $cli->info("Done.");
    } else {
        $cli->error("Nepodařilo se zkopírovat jeden nebo více souborů do cloudu.");
    }
}

exit(0);
