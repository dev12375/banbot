CREATE EXTENSION IF NOT EXISTS timescaledb;

-- ----------------------------
-- Table structure for bottask
-- ----------------------------
DROP TABLE IF EXISTS "public"."bottask";
CREATE TABLE "public"."bottask"
(
    "id"        BIGSERIAL     NOT NULL PRIMARY KEY,
    "mode"      varchar(10)   not null,
    "stg_hash"  varchar(32)   not null,
    "create_at" int8          not null,
    "start_at"  int8          not null,
    "stop_at"   int8          not null,
    "info"      varchar(4096) not null
)
;

-- ----------------------------
-- Table structure for exorder
-- ----------------------------
DROP TABLE IF EXISTS "public"."exorder";
CREATE TABLE "public"."exorder"
(
    "id"         BIGSERIAL    NOT NULL PRIMARY KEY,
    "task_id"    int4         not null,
    "inout_id"   int4         not null,
    "symbol"     varchar(50)  not null,
    "enter"      bool         not null,
    "order_type" varchar(50)  not null,
    "order_id"   varchar(164) not null,
    "side"       varchar(10)  not null,
    "create_at"  int8         not null,
    "price"      float8       not null,
    "average"    float8       not null,
    "amount"     float8       not null,
    "filled"     float8       not null,
    "status"     int2         not null,
    "fee"        float8       not null,
    "fee_type"   varchar(10)  not null,
    "update_at"  int8         not null
);

CREATE INDEX "idx_od_inout_id" ON "public"."exorder" USING btree ("inout_id");
CREATE INDEX "idx_od_status" ON "public"."exorder" USING btree ("status");
CREATE INDEX "idx_od_task_id" ON "public"."exorder" USING btree ("task_id");


-- ----------------------------
-- Table structure for iorder
-- ----------------------------
DROP TABLE IF EXISTS "public"."iorder";
CREATE TABLE "public"."iorder"
(
    "id"          BIGSERIAL     NOT NULL PRIMARY KEY,
    "task_id"     int4          not null,
    "symbol"      varchar(50)   not null,
    "sid"         int4          not null,
    "timeframe"   varchar(5)    not null,
    "short"       bool          not null,
    "status"      int2          not null,
    "enter_tag"   varchar(30)   not null,
    "init_price"  float8        not null,
    "quote_cost"  float8        not null,
    "exit_tag"    varchar(30)   not null,
    "leverage"    int4          not null,
    "enter_at"    int8          not null,
    "exit_at"     int8          not null,
    "strategy"    varchar(20)   not null,
    "stg_ver"     int4          not null,
    "profit_rate" float8        not null,
    "profit"      float8        not null,
    "info"        varchar(1024) not null
);

CREATE INDEX "idx_io_status" ON "public"."iorder" USING btree ("status");
CREATE INDEX "idx_io_task_id" ON "public"."iorder" USING btree ("task_id");


-- ----------------------------
-- Table structure for khole
-- ----------------------------
DROP TABLE IF EXISTS "public"."khole";
CREATE TABLE "public"."khole"
(
    "id"        BIGSERIAL  NOT NULL PRIMARY KEY,
    "sid"       int4       not null,
    "timeframe" varchar(5) not null,
    "start"     int8       not null,
    "stop"      int8       not null
);
CREATE INDEX "idx_khole_sid_tf" ON "public"."khole" USING btree ("sid", "timeframe");
ALTER TABLE "public"."khole"
    ADD CONSTRAINT "idx_khole_sid_tf_start" UNIQUE ("sid", "timeframe", "start");

-- ----------------------------
-- Table structure for kinfo
-- ----------------------------
DROP TABLE IF EXISTS "public"."kinfo";
CREATE TABLE "public"."kinfo"
(
    "sid"       BIGSERIAL  NOT NULL PRIMARY KEY,
    "timeframe" varchar(5) NOT NULL,
    "start"     int8       not null,
    "stop"      int8       not null
);

-- ----------------------------
-- Table structure for kline_1m
-- ----------------------------
DROP TABLE IF EXISTS "public"."kline_1m";
CREATE TABLE "public"."kline_1m"
(
    "sid"    int4 NOT NULL,
    "time"   int8      not null,
    "open"   float8    not null,
    "high"   float8    not null,
    "low"    float8    not null,
    "close"  float8    not null,
    "volume" float8    not null
);
CREATE INDEX "kline_1m_sid" ON "public"."kline_1m" USING btree ("sid");
SELECT create_hypertable('kline_1m', by_range('time', 5184000000));
ALTER TABLE "public"."kline_1m"
    SET (
        timescaledb.compress,
        timescaledb.compress_orderby = 'time DESC',
        timescaledb.compress_segmentby = 'sid'
        );
SELECT add_compression_policy('kline_1m', 5184000000);

-- ----------------------------
-- Table structure for kline_5m
-- ----------------------------
DROP TABLE IF EXISTS "public"."kline_5m";
CREATE TABLE "public"."kline_5m"
(
    "sid"    int4 NOT NULL,
    "time"   int8      not null,
    "open"   float8    not null,
    "high"   float8    not null,
    "low"    float8    not null,
    "close"  float8    not null,
    "volume" float8    not null
);
CREATE INDEX "kline_5m_sid" ON "public"."kline_5m" USING btree ("sid");
SELECT create_hypertable('kline_5m', by_range('time', 7776000000));
ALTER TABLE "public"."kline_5m"
    SET (
        timescaledb.compress,
        timescaledb.compress_orderby = 'time DESC',
        timescaledb.compress_segmentby = 'sid'
        );
SELECT add_compression_policy('kline_5m', 5184000000);

-- ----------------------------
-- Table structure for kline_15m
-- ----------------------------
DROP TABLE IF EXISTS "public"."kline_15m";
CREATE TABLE "public"."kline_15m"
(
    "sid"    int4 NOT NULL,
    "time"   int8      not null,
    "open"   float8    not null,
    "high"   float8    not null,
    "low"    float8    not null,
    "close"  float8    not null,
    "volume" float8    not null
);
CREATE INDEX "kline_15m_sid" ON "public"."kline_15m" USING btree ("sid");
SELECT create_hypertable('kline_15m', by_range('time', 31536000000));
ALTER TABLE "public"."kline_15m"
    SET (
        timescaledb.compress,
        timescaledb.compress_orderby = 'time DESC',
        timescaledb.compress_segmentby = 'sid'
        );
SELECT add_compression_policy('kline_15m', 7776000000);

-- ----------------------------
-- Table structure for kline_1h
-- ----------------------------
DROP TABLE IF EXISTS "public"."kline_1h";
CREATE TABLE "public"."kline_1h"
(
    "sid"    int4 NOT NULL,
    "time"   int8      not null,
    "open"   float8    not null,
    "high"   float8    not null,
    "low"    float8    not null,
    "close"  float8    not null,
    "volume" float8    not null
);
CREATE INDEX "kline_1h_sid" ON "public"."kline_1h" USING btree ("sid");
SELECT create_hypertable('kline_1h', by_range('time', 63072000000));
ALTER TABLE "public"."kline_1h"
    SET (
        timescaledb.compress,
        timescaledb.compress_orderby = 'time DESC',
        timescaledb.compress_segmentby = 'sid'
        );
SELECT add_compression_policy('kline_1h', 15552000000);

-- ----------------------------
-- Table structure for kline_1d
-- ----------------------------
DROP TABLE IF EXISTS "public"."kline_1d";
CREATE TABLE "public"."kline_1d"
(
    "sid"    int4 NOT NULL,
    "time"   int8      not null,
    "open"   float8    not null,
    "high"   float8    not null,
    "low"    float8    not null,
    "close"  float8    not null,
    "volume" float8    not null
);
CREATE INDEX "kline_1d_sid" ON "public"."kline_1d" USING btree ("sid");
SELECT create_hypertable('kline_1d', by_range('time', 94608000000));
ALTER TABLE "public"."kline_1d"
    SET (
        timescaledb.compress,
        timescaledb.compress_orderby = 'time DESC',
        timescaledb.compress_segmentby = 'sid'
        );
SELECT add_compression_policy('kline_1d', 94608000000);

-- ----------------------------
-- Table structure for kline_un
-- ----------------------------
DROP TABLE IF EXISTS "public"."kline_un";
CREATE TABLE "public"."kline_un"
(
    "sid"       BIGSERIAL  NOT NULL PRIMARY KEY,
    "start_ms"  int8       NOT NULL,
    "stop_ms"   int8       NOT NULL,
    "timeframe" varchar(5) NOT NULL,
    "open"      float8     not null,
    "high"      float8     not null,
    "low"       float8     not null,
    "close"     float8     not null,
    "volume"    float8     not null
);
CREATE INDEX "kline_un_sid_tf_idx" ON "public"."kline_un" USING btree ("sid", "timeframe");

-- ----------------------------
-- Table structure for overlay
-- ----------------------------
DROP TABLE IF EXISTS "public"."overlay";
CREATE TABLE "public"."overlay"
(
    "id"        BIGSERIAL NOT NULL PRIMARY KEY,
    "user"      int4      not null,
    "sid"       int4      not null,
    "start_ms"  int8      not null,
    "stop_ms"   int8      not null,
    "tf_msecs"  int4      not null,
    "update_at" int8      not null,
    "data"      text      not null
);
CREATE INDEX "ix_overlay_sid" ON "public"."overlay" USING btree ("sid");
CREATE INDEX "ix_overlay_start_ms" ON "public"."overlay" USING btree ("start_ms");
CREATE INDEX "ix_overlay_stop_ms" ON "public"."overlay" USING btree ("stop_ms");
CREATE INDEX "ix_overlay_user" ON "public"."overlay" USING btree ("user");

-- ----------------------------
-- Table structure for symbol
-- ----------------------------
DROP TABLE IF EXISTS "public"."symbol";
CREATE TABLE "public"."symbol"
(
    "id"        BIGSERIAL   NOT NULL PRIMARY KEY,
    "exchange"  varchar(50) not null,
    "symbol"    varchar(20) not null,
    "market"    varchar(20) not null,
    "list_dt"   int8        not null,
    "delist_dt" int8        not null
);


-- ----------------------------
-- Table structure for users
-- ----------------------------
DROP TABLE IF EXISTS "public"."users";
CREATE TABLE "public"."users"
(
    "id"              BIGSERIAL    NOT NULL PRIMARY KEY,
    "user_name"       varchar(128) not null,
    "avatar"          varchar(256) not null,
    "mobile"          varchar(30)  not null,
    "mobile_verified" bool         not null,
    "email"           varchar(120) not null,
    "email_verified"  bool         not null,
    "pwd_salt"        varchar(128) not null,
    "last_ip"         varchar(64)  not null,
    "create_at"       int8         not null,
    "last_login"      int8         not null,
    "vip_type"        int4         not null,
    "vip_expire_at"   int8         not null,
    "inviter_id"      int4         not null
);
CREATE INDEX "idx_user_mobile" ON "public"."users" USING btree ("mobile");



