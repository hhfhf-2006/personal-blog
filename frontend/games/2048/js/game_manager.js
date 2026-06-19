function GameManager(size, InputManager, Actuator, StorageManager) {
  this.size           = size; // Size of the grid
  this.inputManager   = new InputManager;
  this.storageManager = new StorageManager;
  this.actuator       = new Actuator;

  this.startTiles     = 2;

  this.inputManager.on("move", this.move.bind(this));
  this.inputManager.on("restart", this.restart.bind(this));
  this.inputManager.on("keepPlaying", this.keepPlaying.bind(this));

  this.setup();
}

// Submit score to the blog's backend API.
//
// 工程逻辑（单一备份 + 去重防抖）：
//   1. 去重：本标签页内已确认提交过 >= 当前分数 → 跳过（既防连点，也契合
//      后端"仅当超过历史最高分才更新"的语义）。
//   2. 防并发：相同分数的请求已在途 → 跳过，避免重复结算请求。
//   3. 把待提交的最高分写入唯一备份 key『pendingScore』，供页面卸载时
//      keepalive 兜底（不依赖 gameState 生命周期，不会被 restart 清空）。
//   4. api.post 成功 → 记录水位、清除 pendingScore + gameState、刷新 Best 显示。
//   5. api.post 失败 → 保留 pendingScore，由 pagehide/beforeunload 兜底重试。
GameManager.prototype.submitScore = function () {
  var self = this;

  if (this.score <= 0) return;

  // 未登录或 api 未加载则跳过
  if (typeof api === 'undefined') return;
  try {
    if (!localStorage.getItem('token')) return;
  } catch (e) {
    return;
  }

  // —— 去重：已确认提交过 >= 当前分数 → 无需再提交 ——
  if (this.score <= (this.lastSubmittedScore || 0)) return;

  // —— 防并发：相同分数的请求已在途 → 跳过 ——
  if (this._submitInFlight === this.score) return;
  this._submitInFlight = this.score;

  // 合并上一笔可能失败残留的 pendingScore，始终向后端推送已知最高分；
  // 后端 UPSERT 只保留最大值，重复/低分提交不会污染数据。
  var score = this.score;
  try {
    var prev = JSON.parse(localStorage.getItem('pendingScore') || 'null');
    if (prev && prev.score > score) score = prev.score;
  } catch (e) {}

  // 唯一兜底备份 key：仅在 api.post 成功后清除（失败保留）。
  try {
    localStorage.setItem('pendingScore', JSON.stringify({
      game_name: '2048',
      score: score
    }));
  } catch (e) {}

  api.post('/games/scores', { game_name: '2048', score: score })
    .then(function (res) {
      // 提交成功 → 记录会话最高水位，清除备份和游戏状态
      self.lastSubmittedScore = score;
      self._submitInFlight = 0;
      try { localStorage.removeItem('pendingScore'); } catch (e) {}
      self.storageManager.clearGameState();

      // 用后端返回的确认最高分更新 Best 显示，
      // 确保 2048 页面的 Best Score 与排行榜数据库一致（single source of truth）。
      if (res && res.data && typeof res.data.score === 'number' && res.data.score > 0) {
        var confirmedBest = res.data.score;
        // 更新 localStorage 中的 bestScore（供后续游戏读取）
        if (Number(self.storageManager.getBestScore()) < confirmedBest) {
          self.storageManager.setBestScore(confirmedBest);
        }
        // 直接更新页面上的 Best 显示
        if (self.actuator && self.actuator.bestDisplay) {
          self.actuator.bestDisplay.textContent = confirmedBest.toLocaleString();
        }
      }
    })
    .catch(function () {
      // 提交失败 → 保留 pendingScore，供 pagehide/beforeunload 用 keepalive 兜底重试。
      // 不清除 pendingScore，也不依赖 gameState（restart 后会被覆盖）。
      self._submitInFlight = 0;
      if (typeof showToast !== 'undefined') {
        showToast('分数提交失败，离开页面时将自动重试', 'error');
      }
    });
};

// Restart the game.
//
// 单一备份机制（重启场景）：
//   开新局前先 submitScore() 提交当前局分数。submitScore() 失败时会保留
//   pendingScore（不再被 .catch 清除），即使随后 setup() 覆盖 gameState，
//   pendingScore 仍是有效的卸载兜底来源。无需额外的 pendingRestartScore 层。
//
//   连点防抖：忽略 400ms 内的重复 restart，避免连点 New Game 触发多次结算。
//
// 使用 _freshStart 标志告诉 setup() 跳过旧状态恢复，直接开新局。
GameManager.prototype.restart = function () {
  // —— 连点防抖：400ms 内的重复点击直接忽略 ——
  var now = Date.now();
  if (this._lastRestart && (now - this._lastRestart) < 400) return;
  this._lastRestart = now;

  // 开新局前提交当前分数（覆盖"点击 New Game"这一结束时机）
  this.submitScore();

  this.actuator.continueGame();
  this.steps = 0;
  this.startTime = Date.now();
  this._freshStart = true;
  this.setup();
};

// Keep playing after winning (allows going over 2048)
GameManager.prototype.keepPlaying = function () {
  this.keepPlaying = true;
  this.actuator.continueGame(); // Clear the game won/lost message
};

// Return true if the game is lost, or has won and the user hasn't kept playing
GameManager.prototype.isGameTerminated = function () {
  return this.over || (this.won && !this.keepPlaying);
};

// Set up the game
GameManager.prototype.setup = function () {
  // _freshStart 由 restart() 设置：跳过 localStorage 恢复，直接开新局
  var previousState = this._freshStart ? null : this.storageManager.getGameState();
  this._freshStart = false;

  // Reload the game from a previous game if present
  if (previousState) {
    this.grid        = new Grid(previousState.grid.size,
                                previousState.grid.cells); // Reload grid
    this.score       = previousState.score;
    this.over        = previousState.over;
    this.won         = previousState.won;
    this.keepPlaying = previousState.keepPlaying;
    this.steps       = previousState.steps || 0;
    this.startTime   = previousState.startTime || Date.now();
  } else {
    this.grid        = new Grid(this.size);
    this.score       = 0;
    this.over        = false;
    this.won         = false;
    this.keepPlaying = false;
    this.steps       = 0;
    this.startTime   = Date.now();

    // Add the initial tiles
    this.addStartTiles();
  }

  // Update the actuator
  this.actuate();
};

// Set up the initial tiles to start the game with
GameManager.prototype.addStartTiles = function () {
  for (var i = 0; i < this.startTiles; i++) {
    this.addRandomTile();
  }
};

// Adds a tile in a random position
GameManager.prototype.addRandomTile = function () {
  if (this.grid.cellsAvailable()) {
    var value = Math.random() < 0.9 ? 2 : 4;
    var tile = new Tile(this.grid.randomAvailableCell(), value);

    this.grid.insertTile(tile);
  }
};

// Sends the updated grid to the actuator
GameManager.prototype.actuate = function () {
  if (this.storageManager.getBestScore() < this.score) {
    this.storageManager.setBestScore(this.score);
  }

  // 始终将当前状态持久化到 localStorage：
  //   - 游戏进行中：防止刷新/关闭丢失进度
  //   - 游戏结束：保存最终分数，作为 beforeunload 兜底提交的数据源
  // submitScore() 成功后会自动清除，失败则保留供 beforeunload 重试
  this.storageManager.setGameState(this.serialize());

  if (this.over) {
    this.submitScore();
  }

  this.actuator.actuate(this.grid, {
    score:      this.score,
    over:       this.over,
    won:        this.won,
    bestScore:  this.storageManager.getBestScore(),
    terminated: this.isGameTerminated(),
    steps:      this.steps,
    elapsed:    Math.floor((Date.now() - this.startTime) / 1000)
  });

};

// Represent the current game as an object
GameManager.prototype.serialize = function () {
  return {
    grid:        this.grid.serialize(),
    score:       this.score,
    over:        this.over,
    won:         this.won,
    keepPlaying: this.keepPlaying,
    steps:       this.steps,
    startTime:   this.startTime
  };
};

// Save all tile positions and remove merger info
GameManager.prototype.prepareTiles = function () {
  this.grid.eachCell(function (x, y, tile) {
    if (tile) {
      tile.mergedFrom = null;
      tile.savePosition();
    }
  });
};

// Move a tile and its representation
GameManager.prototype.moveTile = function (tile, cell) {
  this.grid.cells[tile.x][tile.y] = null;
  this.grid.cells[cell.x][cell.y] = tile;
  tile.updatePosition(cell);
};

// Move tiles on the grid in the specified direction
GameManager.prototype.move = function (direction) {
  // 0: up, 1: right, 2: down, 3: left
  var self = this;

  if (this.isGameTerminated()) return; // Don't do anything if the game's over

  var cell, tile;

  var vector     = this.getVector(direction);
  var traversals = this.buildTraversals(vector);
  var moved      = false;

  // Save the current tile positions and remove merger information
  this.prepareTiles();

  // Traverse the grid in the right direction and move tiles
  traversals.x.forEach(function (x) {
    traversals.y.forEach(function (y) {
      cell = { x: x, y: y };
      tile = self.grid.cellContent(cell);

      if (tile) {
        var positions = self.findFarthestPosition(cell, vector);
        var next      = self.grid.cellContent(positions.next);

        // Only one merger per row traversal?
        if (next && next.value === tile.value && !next.mergedFrom) {
          var merged = new Tile(positions.next, tile.value * 2);
          merged.mergedFrom = [tile, next];

          self.grid.insertTile(merged);
          self.grid.removeTile(tile);

          // Converge the two tiles' positions
          tile.updatePosition(positions.next);

          // Update the score
          self.score += merged.value;

          // The mighty 2048 tile
          if (merged.value === 2048) self.won = true;
        } else {
          self.moveTile(tile, positions.farthest);
        }

        if (!self.positionsEqual(cell, tile)) {
          moved = true; // The tile moved from its original cell!
        }
      }
    });
  });

  if (moved) {
    this.steps++;
    this.addRandomTile();

    if (!this.movesAvailable()) {
      this.over = true; // Game over!
    }

    this.actuate();
  }
};

// Get the vector representing the chosen direction
GameManager.prototype.getVector = function (direction) {
  // Vectors representing tile movement
  var map = {
    0: { x: 0,  y: -1 }, // Up
    1: { x: 1,  y: 0 },  // Right
    2: { x: 0,  y: 1 },  // Down
    3: { x: -1, y: 0 }   // Left
  };

  return map[direction];
};

// Build a list of positions to traverse in the right order
GameManager.prototype.buildTraversals = function (vector) {
  var traversals = { x: [], y: [] };

  for (var pos = 0; pos < this.size; pos++) {
    traversals.x.push(pos);
    traversals.y.push(pos);
  }

  // Always traverse from the farthest cell in the chosen direction
  if (vector.x === 1) traversals.x = traversals.x.reverse();
  if (vector.y === 1) traversals.y = traversals.y.reverse();

  return traversals;
};

GameManager.prototype.findFarthestPosition = function (cell, vector) {
  var previous;

  // Progress towards the vector direction until an obstacle is found
  do {
    previous = cell;
    cell     = { x: previous.x + vector.x, y: previous.y + vector.y };
  } while (this.grid.withinBounds(cell) &&
           this.grid.cellAvailable(cell));

  return {
    farthest: previous,
    next: cell // Used to check if a merge is required
  };
};

GameManager.prototype.movesAvailable = function () {
  return this.grid.cellsAvailable() || this.tileMatchesAvailable();
};

// Check for available matches between tiles (more expensive check)
GameManager.prototype.tileMatchesAvailable = function () {
  var self = this;

  var tile;

  for (var x = 0; x < this.size; x++) {
    for (var y = 0; y < this.size; y++) {
      tile = this.grid.cellContent({ x: x, y: y });

      if (tile) {
        for (var direction = 0; direction < 4; direction++) {
          var vector = self.getVector(direction);
          var cell   = { x: x + vector.x, y: y + vector.y };

          var other  = self.grid.cellContent(cell);

          if (other && other.value === tile.value) {
            return true; // These two tiles can be merged
          }
        }
      }
    }
  }

  return false;
};

GameManager.prototype.positionsEqual = function (first, second) {
  return first.x === second.x && first.y === second.y;
};
