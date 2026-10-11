<?php
/**
 * Constants for OMDB scripts
 * php version 8.2
 *
 * @category Helper
 * @package  None
 * @author   Fred Brooker <git@gscloud.cz>
 * @license  MIT https://gscloud.cz/LICENSE
 * @link     None
 */

define('CACHE', './cache.json');
define('CACHE2', './cache2.json');

define('DB', 'games.txt');
define('SCORES', './go/games.csv');

define('EX1', './stats-NO-SCORES.txt');
define('EX2', './stats-WORST-50.txt');
define('EX4', './stats-ALL-by-SCORE.txt');
define('EX5', './stats-NO-SCORES-NAMES.txt');
define('EX7', './stats-ALL-by-SIZE.txt');

define('FS', '/home/mxdpeep/DS3/');

define('SF', RecursiveIteratorIterator::SELF_FIRST);
define('SKDS', RecursiveDirectoryIterator::SKIP_DOTS);

// path stub => iterator depth
$DF = [
    "GAME/" => 0,
    "GAME GOG/" => 0,
    "GAME NODRM/" => 0,
];
