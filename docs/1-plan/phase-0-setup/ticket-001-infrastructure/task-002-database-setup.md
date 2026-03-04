# TASK: Database Infrastructure Setup

**Ticket**: Infrastructure Setup  
**Priority**: P0 (Critical)  
**Assignee**: Database Engineer + DevOps Engineer  
**Estimate**: 3 days  
**Dependencies**: AWS Cloud Setup  

## 📋 TASK OVERVIEW

**Objective**: Setup PostgreSQL RDS និង Redis ElastiCache for production  
**Success Criteria**: Database clusters operational with backup និង monitoring  

---

## 🎯 **REQUIREMENTS:**

### **PostgreSQL RDS Setup:**
- [ ] Multi-AZ RDS PostgreSQL 14+ instance
- [ ] Read replicas for performance scaling
- [ ] Automated backup និង point-in-time recovery
- [ ] Parameter groups for optimization

### **Redis ElastiCache:**
- [ ] Redis cluster mode enabled
- [ ] Multi-AZ deployment for high availability
- [ ] Cache parameter groups optimization
- [ ] Encryption at rest និង in transit

### **Database Security:**
- [ ] VPC security groups និង subnet groups
- [ ] Database encryption at rest
- [ ] SSL/TLS connection enforcement
- [ ] Parameter store for secrets management

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **PostgreSQL RDS Configuration:**
```hcl
# terraform/database/rds.tf
resource "aws_db_instance" "shortlink_postgres" {
  identifier = "shortlink-postgres-${var.environment}"
  
  # Engine Configuration
  engine         = "postgres"
  engine_version = "14.9"
  instance_class = var.environment == "production" ? "db.r6g.xlarge" : "db.t3.medium"
  
  # Storage Configuration
  allocated_storage     = var.environment == "production" ? 100 : 20
  max_allocated_storage = var.environment == "production" ? 1000 : 100
  storage_type         = "gp3"
  storage_encrypted    = true
  
  # Database Configuration
  db_name  = "shortlink"
  username = "shortlink_admin"
  password = random_password.db_password.result
  
  # Network Configuration
  db_subnet_group_name   = aws_db_subnet_group.shortlink_subnet_group.name
  vpc_security_group_ids = [aws_security_group.rds_sg.id]
  
  # High Availability
  multi_az               = var.environment == "production" ? true : false
  backup_retention_period = var.environment == "production" ? 30 : 7
  backup_window          = "03:00-04:00"
  maintenance_window     = "sun:04:00-sun:05:00"
  
  # Monitoring
  performance_insights_enabled = true
  monitoring_interval         = 60
  monitoring_role_arn        = aws_iam_role.rds_monitoring.arn
  
  # Security
  deletion_protection = var.environment == "production" ? true : false
  skip_final_snapshot = var.environment == "production" ? false : true
  
  tags = {
    Name        = "shortlink-postgres"
    Environment = var.environment
  }
}

# Read Replica for Production
resource "aws_db_instance" "shortlink_postgres_replica" {
  count = var.environment == "production" ? 2 : 0
  
  identifier = "shortlink-postgres-replica-${count.index + 1}"
  replicate_source_db = aws_db_instance.shortlink_postgres.identifier
  
  instance_class = "db.r6g.large"
  
  tags = {
    Name        = "shortlink-postgres-replica-${count.index + 1}"
    Environment = var.environment
  }
}
```

### **Redis ElastiCache Cluster:**
```hcl
# terraform/database/redis.tf
resource "aws_elasticache_replication_group" "shortlink_redis" {
  replication_group_id       = "shortlink-redis-${var.environment}"
  description                = "Redis cluster for ShortLink application"
  
  # Engine Configuration
  engine               = "redis"
  engine_version       = "7.0"
  node_type           = var.environment == "production" ? "cache.r6g.xlarge" : "cache.t3.medium"
  
  # Cluster Configuration
  num_cache_clusters         = var.environment == "production" ? 3 : 1
  port                      = 6379
  parameter_group_name      = aws_elasticache_parameter_group.shortlink_redis.name
  
  # Network Configuration
  subnet_group_name = aws_elasticache_subnet_group.shortlink_cache_subnet.name
  security_group_ids = [aws_security_group.redis_sg.id]
  
  # High Availability
  multi_az_enabled           = var.environment == "production" ? true : false
  automatic_failover_enabled = var.environment == "production" ? true : false
  
  # Security
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                = random_password.redis_auth.result
  
  # Backup
  snapshot_retention_limit = var.environment == "production" ? 7 : 1
  snapshot_window         = "03:00-05:00"
  
  tags = {
    Name        = "shortlink-redis"
    Environment = var.environment
  }
}

# Redis Parameter Group for Optimization
resource "aws_elasticache_parameter_group" "shortlink_redis" {
  name   = "shortlink-redis-params-${var.environment}"
  family = "redis7.x"
  
  # Memory Management
  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }
  
  # Performance Tuning
  parameter {
    name  = "timeout"
    value = "300"
  }
  
  parameter {
    name  = "tcp-keepalive"
    value = "60"
  }
}
```

### **Database Initialization Script:**
```sql
-- database/migrations/001_initial_schema.sql
-- ShortLink Database Schema

-- Enable Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For text search
CREATE EXTENSION IF NOT EXISTS "btree_gin"; -- For composite indexes

-- Users Table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    
    -- OAuth Fields
    google_id VARCHAR(255),
    github_id VARCHAR(255),
    
    -- Account Status
    email_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    account_type VARCHAR(20) DEFAULT 'free' CHECK (account_type IN ('free', 'pro', 'business', 'enterprise')),
    
    -- Preferences
    timezone VARCHAR(50) DEFAULT 'UTC',
    locale VARCHAR(10) DEFAULT 'en',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE
);

-- Short URLs Table (Core Table - Highest Priority)
CREATE TABLE short_urls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- URL Fields
    short_code VARCHAR(10) UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    title VARCHAR(500),
    description TEXT,
    
    -- Customization
    custom_alias VARCHAR(50),
    domain VARCHAR(100) DEFAULT 'sh.ly',
    
    -- Settings
    is_active BOOLEAN DEFAULT TRUE,
    password_protected BOOLEAN DEFAULT FALSE,
    password_hash VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Statistics (Denormalized for Performance)
    total_clicks INTEGER DEFAULT 0,
    unique_clicks INTEGER DEFAULT 0,
    last_clicked_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- URL Clicks Analytics Table
CREATE TABLE url_clicks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_url_id UUID REFERENCES short_urls(id) ON DELETE CASCADE,
    
    -- Request Information
    ip_address INET NOT NULL,
    user_agent TEXT,
    referer TEXT,
    
    -- Geographic Data
    country VARCHAR(2),
    region VARCHAR(100),
    city VARCHAR(100),
    
    -- Device Information
    device_type VARCHAR(20), -- mobile, desktop, tablet
    device_brand VARCHAR(50),
    device_model VARCHAR(100),
    browser VARCHAR(50),
    browser_version VARCHAR(50),
    os VARCHAR(50),
    os_version VARCHAR(50),
    
    -- Timestamp
    clicked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for Performance
CREATE INDEX idx_short_urls_user_id ON short_urls(user_id);
CREATE INDEX idx_short_urls_short_code ON short_urls(short_code);
CREATE INDEX idx_short_urls_created_at ON short_urls(created_at);
CREATE INDEX idx_short_urls_active ON short_urls(is_active) WHERE is_active = TRUE;

CREATE INDEX idx_url_clicks_short_url_id ON url_clicks(short_url_id);
CREATE INDEX idx_url_clicks_clicked_at ON url_clicks(clicked_at);
CREATE INDEX idx_url_clicks_ip_address ON url_clicks(ip_address);
CREATE INDEX idx_url_clicks_country ON url_clicks(country);

-- Composite Indexes for Analytics Queries
CREATE INDEX idx_url_clicks_analytics ON url_clicks(short_url_id, clicked_at, country, device_type);
CREATE INDEX idx_short_urls_user_stats ON short_urls(user_id, created_at, total_clicks);

-- Function to Update URL Click Statistics
CREATE OR REPLACE FUNCTION update_url_click_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE short_urls 
    SET 
        total_clicks = total_clicks + 1,
        last_clicked_at = NOW(),
        updated_at = NOW()
    WHERE id = NEW.short_url_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to Auto-Update Statistics
CREATE TRIGGER trigger_update_url_click_stats
    AFTER INSERT ON url_clicks
    FOR EACH ROW
    EXECUTE FUNCTION update_url_click_stats();

-- Function to Generate Short Codes
CREATE OR REPLACE FUNCTION generate_short_code(length INTEGER DEFAULT 6)
RETURNS VARCHAR AS $$
DECLARE
    chars VARCHAR := 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789';
    result VARCHAR := '';
    i INTEGER := 0;
BEGIN
    FOR i IN 1..length LOOP
        result := result || substr(chars, floor(random() * length(chars) + 1)::integer, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

---

## 🔧 **DATABASE OPTIMIZATION:**

### **Performance Tuning Parameters:**
```sql
-- PostgreSQL Performance Settings
-- postgresql.conf optimizations

# Memory Settings
shared_buffers = '256MB'              # 25% of available RAM
effective_cache_size = '1GB'          # 75% of available RAM
work_mem = '4MB'                      # Per operation memory

# Checkpoint Settings
checkpoint_completion_target = 0.7
wal_buffers = '16MB'
checkpoint_segments = 32

# Query Planner
random_page_cost = 1.1                # SSD optimization
effective_io_concurrency = 200        # SSD optimization

# Connection Settings
max_connections = 200
shared_preload_libraries = 'pg_stat_statements'

# Logging
log_statement = 'ddl'
log_checkpoints = on
log_connections = on
log_disconnections = on
```

### **Redis Configuration:**
```redis
# redis.conf optimizations

# Memory Management
maxmemory-policy allkeys-lru
maxmemory 512mb

# Persistence (for production)
save 900 1     # Save if at least 1 key changed in 900 seconds
save 300 10    # Save if at least 10 keys changed in 300 seconds
save 60 10000  # Save if at least 10000 keys changed in 60 seconds

# Network
timeout 300
tcp-keepalive 60

# Security
requirepass "your-secure-redis-password"
```

---

## ✅ **ACCEPTANCE CRITERIA:**
- [ ] PostgreSQL RDS instance operational với read replicas
- [ ] Redis cluster accessible និង performing well  
- [ ] Database schema migrations completed successfully
- [ ] Automated backups configured និង tested
- [ ] Monitoring និង alerting active
- [ ] Performance baselines established
- [ ] Security configurations validated
- [ ] Connection pooling configured

---

## 🧪 **TESTING CHECKLIST:**
- [ ] Database connectivity from application servers
- [ ] Read replica lag monitoring
- [ ] Redis cluster failover testing
- [ ] Backup និង restore validation
- [ ] Performance benchmarking
- [ ] Security penetration testing

---

**Completion Date**: _________  
**Review By**: Database Lead + DevOps Lead  
**Next Task**: Development Environment Setup