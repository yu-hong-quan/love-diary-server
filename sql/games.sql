-- 游戏模块：五子棋 + 丢骰子比大小
-- PostgreSQL，可重复执行（仅建表，不删数据）

CREATE TABLE IF NOT EXISTS game_sessions (
    id               SERIAL PRIMARY KEY,
    game_type        VARCHAR(20)  NOT NULL,  -- gomoku | dice
    status           VARCHAR(20)  NOT NULL DEFAULT 'playing',  -- playing | finished | draw
    player1_name     VARCHAR(50)  NOT NULL,
    player2_name     VARCHAR(50)  NOT NULL,
    current_turn     SMALLINT     NOT NULL DEFAULT 1,  -- 1 或 2
    winner           SMALLINT     NOT NULL DEFAULT 0,  -- 0 未分出 | 1 | 2 | 3 和局
    board_size       SMALLINT     NOT NULL DEFAULT 15, -- 五子棋棋盘边长
    player1_score    INT          NOT NULL DEFAULT 0,  -- 骰子：胜场数
    player2_score    INT          NOT NULL DEFAULT 0,
    target_wins      INT          NOT NULL DEFAULT 1,  -- 骰子：先赢几局结束
    dice_phase       VARCHAR(20),                     -- roll_p1 | roll_p2（仅 dice）
    player1_last_roll SMALLINT,
    player2_last_roll SMALLINT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_game_sessions_type_status ON game_sessions (game_type, status);
CREATE INDEX IF NOT EXISTS idx_game_sessions_created ON game_sessions (created_at DESC);

CREATE TABLE IF NOT EXISTS gomoku_moves (
    id         SERIAL PRIMARY KEY,
    session_id INT          NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    player     SMALLINT     NOT NULL,  -- 1 黑 | 2 白
    row        SMALLINT     NOT NULL,
    col        SMALLINT     NOT NULL,
    move_no    INT          NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, row, col),
    UNIQUE (session_id, move_no)
);

CREATE INDEX IF NOT EXISTS idx_gomoku_moves_session ON gomoku_moves (session_id, move_no);

CREATE TABLE IF NOT EXISTS dice_rounds (
    id           SERIAL PRIMARY KEY,
    session_id   INT       NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    round_no     INT       NOT NULL,
    player1_roll SMALLINT  NOT NULL CHECK (player1_roll BETWEEN 1 AND 6),
    player2_roll SMALLINT  NOT NULL CHECK (player2_roll BETWEEN 1 AND 6),
    winner       SMALLINT  NOT NULL,  -- 1 | 2 | 3 和
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, round_no)
);

CREATE INDEX IF NOT EXISTS idx_dice_rounds_session ON dice_rounds (session_id, round_no);
