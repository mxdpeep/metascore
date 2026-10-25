<?php
/**
 * Export various data from cache
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

$s = $t = 0;
$cache = $sizes = $scores = [];

// read from cache
if (file_exists(CACHE) && is_readable(CACHE)) {
    $cache = json_decode(@file_get_contents(CACHE) ?: '', true);
    if (is_array($cache)) {
        if ($t = count($cache)) {
            @unlink(EX1);
            @unlink(EX5);

            // iterate
            foreach ($cache as $h => $v) {
                if (!array_key_exists('name', $v)) {
                    continue;
                }
                if (!array_key_exists('path', $v)) {
                    continue;
                }
                $score = null;
                $name = $v['name'];
                $path = $v['path'];
                if (is_numeric($name)) {
                    continue;
                }
                $size = isset($v['size']) ? $v['size'] : null;
                if ($size) {
                    $sizes[$h] = $size;
                }
                // process rating
                $score = null;
                if (strpos($name, '%') !== false) {
                    preg_match('/ \[(\d+)%]/', $name, $score);
                    if (!isset($score[1])) {
                        continue;
                    }
                    $score = (int) $score[1];
                    if ($score !== null && $score > 0) {
                        $scores[$h] = $score;
                    }
                } else {
                    // export missing ratings
                    file_put_contents(EX1, "{$path}{$name}\n", FILE_APPEND | LOCK_EX); // phpcs:ignore
                    file_put_contents(EX5, "{$name}\n", FILE_APPEND | LOCK_EX);
                    $s++;
                }
            }

            // LOWEST RATINGS
            if (is_array($scores)) {
                $c = 0;
                asort($scores);
                @unlink(EX2);
                $cli->bold("\nLOWEST SCORE");
                foreach ($scores as $k => $v) {
                    $c++;
                    $n = "{$cache[$k]['path']}{$cache[$k]['name']}";
                    if ($c <= 20) {
                        $fc = sprintf("%02d", $c);
                        echo "{$fc}. {$n}\n";
                    }
                    file_put_contents(EX2, "{$c}. {$n}\n", FILE_APPEND | LOCK_EX);
                    if ($c >= 50) {
                        break;
                    }
                }
            }

            // HIGHEST RATINGS
            if (is_array($scores)) {
                $c = 0;
                arsort($scores);
                $cli->bold("\nHIGHEST SCORE");
                foreach ($scores as $k => $v) {
                    $c++;
                    $n = "{$cache[$k]['path']}{$cache[$k]['name']}";
                    if ($c <= 20) {
                        $fc = sprintf("%02d", $c);
                        echo "{$fc}. {$n}\n";
                    }
                    if ($c > 20) {
                        break;
                    }
                }
            }

            // ALL BY RATING
            if (is_array($scores)) {
                $c = 0;
                arsort($scores);
                @unlink(EX4);
                foreach ($scores as $k => $v) {
                    $c++;
                    $n = "{$cache[$k]['path']}{$cache[$k]['name']}";
                    $sz = '';
                    if (isset($cache[$k]['size'])) {
                        $sz = "{$cache[$k]['size']}GB";
                    }
                    file_put_contents(
                        EX4,
                        "{$c}. {$n} {$sz}\n",
                        FILE_APPEND | LOCK_EX
                    );
                }
            }

            // ALL BY SIZE
            if (is_array($sizes)) {
                $c = 0;
                arsort($sizes);
                @unlink(EX7);
                foreach ($sizes as $k => $v) {
                    $c++;
                    $n = "{$cache[$k]['path']}{$cache[$k]['name']}";
                    file_put_contents(
                        EX7,
                        "{$c}. {$n} {$cache[$k]['size']}GB\n",
                        FILE_APPEND | LOCK_EX
                    );
                }
            }

            // BIGGEST FOLDERS
            if (is_array($sizes) && count($sizes)) {
                $c = 0;
                arsort($sizes);
                $cli->bold("\nBIGGEST");
                foreach ($sizes as $k => $v) {
                    $c++;
                    $n = "{$cache[$k]['path']}{$cache[$k]['name']}";
                    if ($c <= 20) {
                        $fc = sprintf("%02d", $c);
                        echo "{$fc}. {$n} {$cache[$k]['size']}GB\n";
                    }
                    if ($c > 20) {
                        break;
                    }
                }
            }

            $total = count($sizes);
            $cli->out("\n<green>Total games: <bold>$total</bold> items</green>");
            $cli->out("<green>Missing ratings: <bold>$s</bold> items</green>");
            $cli->out("<green>Games rated: <bold>" . (100 - floor($s * 100 / count($cache))) . " %</bold></green>\n"); // phpcs:ignore
            exit(0);
        }
    }
}

$cli->error("ERROR: Cache is empty!");
exit(1);
