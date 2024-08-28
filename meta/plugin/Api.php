<?php

/* Copyright 2024 Juan B. Rodriguez
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *   
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

function controlrd_log($m) {
	global $plugin;
  shell_exec("/usr/bin/logger"." ".escapeshellarg($m)." -t controlrd");
}

$socket_path = "/var/run/controlrd-api.sock";
$action = $_POST['action'] ?? '';
$params = !empty($_POST['params']) ? json_decode($_POST['params'], true) : new stdClass();

$decoded = json_decode($_POST['params']);

controlrd_log("received params {$_POST['params']}");
/*controlrd_log("received params {$decoded} "." and ${params}");*/

$socket = fsockopen("unix://{$socket_path}", -1, $errno, $errstr, 30);
if (!$socket) {
  http_response_code(503);
  $reply = [
    'data' => null,
    'error' => $errstr,
  ];
  controlrd_log("service unavailable ${errstr}");
  echo json_encode($reply);
  exit;
}

// send data to the golang server
$data = [
    'action' => $action,
    'params' => $params,
];
$json_data = json_encode($data) . "\n";

fwrite($socket, $json_data);

// read the response from the golang server
$response = '';
while (!feof($socket)) {
    $line = fgets($socket, 1024);
    $response .= trim($line);
}

// Close the socket
fclose($socket);

// Send the response back to the mobile app (if needed)
echo json_encode($response);

// Set appropriate HTTP headers and status code
http_response_code(200);
header('Content-Type: application/json');
