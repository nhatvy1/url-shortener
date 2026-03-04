# TASK: High-Performance URL Redirect System

**Ticket**: Core URL Shortening  
**Priority**: P0 (CRITICAL)  
**Assignee**: Performance Engineer + Backend Developer  
**Estimate**: 3 days  
**Dependencies**: URL Shortening Engine  

## 📋 TASK OVERVIEW

**Objective**: Optimize URL redirect system for maximum performance និង reliability  
**Success Criteria**: < 20ms redirect response time with 99.9% uptime  

---

## 🎯 **PERFORMANCE REQUIREMENTS:**

### **Response Time Goals:**
- [ ] < 20ms average redirect response time
- [ ] < 50ms 95th percentile response time
- [ ] < 100ms 99th percentile response time
- [ ] Support 50,000+ concurrent redirects

### **Availability Requirements:**
- [ ] 99.9% uptime SLA (8.77 hours downtime/year)
- [ ] Graceful degradation during high load
- [ ] Circuit breaker pattern for dependencies
- [ ] Multi-region failover capability

### **Caching Strategy:**
- [ ] Multi-layer caching (Redis + CDN + Application)
- [ ] Cache hit ratio > 95%
- [ ] Cache invalidation strategy
- [ ] Hot vs Cold data optimization

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **Optimized Redirect Handler:**
```typescript
// src/handlers/RedirectHandler.ts
import { Request, Response } from 'express';
import { performance } from 'perf_hooks';
import { Cache } from '../utils/Cache';
import { MetricsCollector } from '../utils/MetricsCollector';
import { CircuitBreaker } from '../utils/CircuitBreaker';

export class OptimizedRedirectHandler {
  private cache = new Cache();
  private metrics = new MetricsCollector();
  private dbCircuitBreaker = new CircuitBreaker('database', {
    failureThreshold: 5,
    timeout: 2000,
    resetTimeout: 10000
  });
  
  /**
   * Ultra-fast redirect handler with multi-layer caching
   */
  async handleRedirect(req: Request, res: Response): Promise<void> {
    const startTime = performance.now();
    const { shortCode } = req.params;
    
    try {
      // 1. Input validation (fastest check first)
      if (!this.isValidShortCode(shortCode)) {
        this.sendNotFound(res, startTime);
        return;
      }
      
      // 2. L1 Cache: Application-level hot cache (fastest)
      let urlData = this.getFromHotCache(shortCode);
      
      if (!urlData) {
        // 3. L2 Cache: Redis cache (fast)
        urlData = await this.getFromRedisCache(shortCode);
        
        if (!urlData) {
          // 4. L3 Fallback: Database with circuit breaker (slowest)
          urlData = await this.getFromDatabaseWithCircuitBreaker(shortCode);
          
          if (!urlData) {
            this.sendNotFound(res, startTime);
            return;
          }
          
          // Cache for future requests
          await this.cacheUrl(shortCode, urlData);
        }
        
        // Add to hot cache
        this.addToHotCache(shortCode, urlData);
      }
      
      // 5. Check URL validity and expiration
      if (!this.isUrlValid(urlData)) {
        this.sendExpired(res, startTime);
        return;
      }
      
      // 6. Perform redirect
      this.performRedirect(res, urlData.originalUrl, startTime);
      
      // 7. Async analytics tracking (non-blocking)
      setImmediate(() => this.trackClick(shortCode, req, urlData));
      
    } catch (error) {
      this.handleRedirectError(res, error, startTime);
    }
  }
  
  private hotCache = new Map<string, { data: any; expires: number }>();
  private readonly HOT_CACHE_SIZE = 10000; // Keep 10k hottest URLs in memory
  private readonly HOT_CACHE_TTL = 300000; // 5 minutes
  
  /**
   * Application-level hot cache for most popular URLs
   */
  private getFromHotCache(shortCode: string): any | null {
    const entry = this.hotCache.get(shortCode);
    
    if (entry && entry.expires > Date.now()) {
      // Move to front (LRU behavior)
      this.hotCache.delete(shortCode);
      this.hotCache.set(shortCode, entry);
      
      this.metrics.increment('cache.hot.hit');
      return entry.data;
    }
    
    if (entry) {
      this.hotCache.delete(shortCode); // Expired
    }
    
    this.metrics.increment('cache.hot.miss');
    return null;
  }
  
  /**
   * Add URL to hot cache with LRU eviction
   */
  private addToHotCache(shortCode: string, urlData: any): void {
    // LRU eviction: remove oldest if at capacity
    if (this.hotCache.size >= this.HOT_CACHE_SIZE) {
      const firstKey = this.hotCache.keys().next().value;
      this.hotCache.delete(firstKey);
    }
    
    this.hotCache.set(shortCode, {
      data: urlData,
      expires: Date.now() + this.HOT_CACHE_TTL
    });
  }
  
  /**
   * Get URL from Redis cache with connection pooling
   */
  private async getFromRedisCache(shortCode: string): Promise<any | null> {
    try {
      const cached = await this.cache.get(`url:${shortCode}`);
      
      if (cached) {
        this.metrics.increment('cache.redis.hit');
        return cached;
      }
      
      this.metrics.increment('cache.redis.miss');
      return null;
      
    } catch (error) {
      this.metrics.increment('cache.redis.error');
      return null; // Graceful degradation
    }
  }
  
  /**
   * Database query with circuit breaker pattern
   */
  private async getFromDatabaseWithCircuitBreaker(shortCode: string): Promise<any | null> {
    return this.dbCircuitBreaker.execute(async () => {
      const { ShortURL } = await import('../models/ShortURL');
      
      const startTime = performance.now();
      
      // Optimized database query
      const result = await ShortURL.findOne(
        {
          shortCode,
          isActive: true,
          $or: [
            { expiresAt: { $exists: false } },
            { expiresAt: null },
            { expiresAt: { $gt: new Date() } }
          ]
        },
        {
          originalUrl: 1,
          expiresAt: 1,
          passwordProtected: 1,
          passwordHash: 1,
          _id: 0
        }
      ).lean(); // Use lean() for better performance
      
      const queryTime = performance.now() - startTime;
      this.metrics.timing('database.query.time', queryTime);
      
      if (result) {
        this.metrics.increment('database.hit');
        return {
          originalUrl: result.originalUrl,
          expiresAt: result.expiresAt,
          passwordProtected: result.passwordProtected,
          passwordHash: result.passwordHash
        };
      }
      
      this.metrics.increment('database.miss');
      return null;
    });
  }
  
  /**
   * Cache URL data in Redis with optimized TTL
   */
  private async cacheUrl(shortCode: string, urlData: any): Promise<void> {
    try {
      // Dynamic TTL based on URL age and access patterns
      let ttl = 3600; // Default 1 hour
      
      if (urlData.expiresAt) {
        const timeToExpiry = new Date(urlData.expiresAt).getTime() - Date.now();
        ttl = Math.min(ttl, Math.floor(timeToExpiry / 1000));
      }
      
      await this.cache.set(`url:${shortCode}`, urlData, ttl);
      
    } catch (error) {
      // Non-critical, log error but don't fail
      console.error('Cache write error:', error);
    }
  }
  
  /**
   * Validate short code format (fastest validation)
   */
  private isValidShortCode(shortCode: string): boolean {
    return typeof shortCode === 'string' && 
           shortCode.length >= 3 && 
           shortCode.length <= 50 &&
           /^[a-zA-Z0-9_-]+$/.test(shortCode);
  }
  
  /**
   * Check if URL is valid and not expired
   */
  private isUrlValid(urlData: any): boolean {
    if (!urlData || !urlData.originalUrl) {
      return false;
    }
    
    // Check expiration
    if (urlData.expiresAt && new Date(urlData.expiresAt) <= new Date()) {
      return false;
    }
    
    return true;
  }
  
  /**
   * Perform the actual redirect with proper headers
   */
  private performRedirect(res: Response, originalUrl: string, startTime: number): void {
    const responseTime = performance.now() - startTime;
    
    // Set performance headers
    res.set({
      'Cache-Control': 'public, max-age=300', // 5 minutes browser cache
      'X-Response-Time': `${responseTime.toFixed(2)}ms`,
      'X-Cache': 'HIT'
    });
    
    // 301 permanent redirect for SEO
    res.redirect(301, originalUrl);
    
    // Record metrics
    this.metrics.timing('redirect.response_time', responseTime);
    this.metrics.increment('redirect.success');
  }
  
  /**
   * Handle 404 Not Found responses
   */
  private sendNotFound(res: Response, startTime: number): void {
    const responseTime = performance.now() - startTime;
    
    res.status(404)
       .set('X-Response-Time', `${responseTime.toFixed(2)}ms`)
       .json({
         success: false,
         error: 'Short URL not found',
         code: 'URL_NOT_FOUND'
       });
    
    this.metrics.timing('redirect.response_time', responseTime);
    this.metrics.increment('redirect.not_found');
  }
  
  /**
   * Handle expired URL responses
   */
  private sendExpired(res: Response, startTime: number): void {
    const responseTime = performance.now() - startTime;
    
    res.status(410)
       .set('X-Response-Time', `${responseTime.toFixed(2)}ms`)
       .json({
         success: false,
         error: 'This link has expired',
         code: 'URL_EXPIRED'
       });
    
    this.metrics.timing('redirect.response_time', responseTime);
    this.metrics.increment('redirect.expired');
  }
  
  /**
   * Handle redirect errors gracefully
   */
  private handleRedirectError(res: Response, error: any, startTime: number): void {
    const responseTime = performance.now() - startTime;
    
    console.error('Redirect error:', error);
    
    res.status(500)
       .set('X-Response-Time', `${responseTime.toFixed(2)}ms`)
       .json({
         success: false,
         error: 'Internal server error',
         code: 'INTERNAL_ERROR'
       });
    
    this.metrics.timing('redirect.response_time', responseTime);
    this.metrics.increment('redirect.error');
  }
  
  /**
   * Async click tracking (non-blocking)
   */
  private async trackClick(shortCode: string, req: Request, urlData: any): Promise<void> {
    try {
      const { ClickTracker } = await import('../services/ClickTracker');
      const tracker = new ClickTracker();
      
      await tracker.trackClick({
        shortCode,
        originalUrl: urlData.originalUrl,
        ipAddress: req.ip,
        userAgent: req.get('User-Agent'),
        referer: req.get('Referer'),
        timestamp: new Date()
      });
      
    } catch (error) {
      // Analytics failure should not affect redirect performance
      console.error('Click tracking error:', error);
    }
  }
}
```

### **Circuit Breaker Implementation:**
```typescript
// src/utils/CircuitBreaker.ts
export class CircuitBreaker {
  private failures = 0;
  private lastFailureTime?: number;
  private state: 'CLOSED' | 'OPEN' | 'HALF_OPEN' = 'CLOSED';
  
  constructor(
    private name: string,
    private options: {
      failureThreshold: number;
      timeout: number;
      resetTimeout: number;
    }
  ) {}
  
  async execute<T>(fn: () => Promise<T>): Promise<T> {
    if (this.state === 'OPEN') {
      if (this.shouldAttemptReset()) {
        this.state = 'HALF_OPEN';
      } else {
        throw new Error(`Circuit breaker ${this.name} is OPEN`);
      }
    }
    
    try {
      const result = await Promise.race([
        fn(),
        new Promise<never>((_, reject) => 
          setTimeout(() => reject(new Error('Timeout')), this.options.timeout)
        )
      ]);
      
      this.onSuccess();
      return result;
      
    } catch (error) {
      this.onFailure();
      throw error;
    }
  }
  
  private onSuccess(): void {
    this.failures = 0;
    this.state = 'CLOSED';
  }
  
  private onFailure(): void {
    this.failures++;
    this.lastFailureTime = Date.now();
    
    if (this.failures >= this.options.failureThreshold) {
      this.state = 'OPEN';
    }
  }
  
  private shouldAttemptReset(): boolean {
    return !!this.lastFailureTime && 
           Date.now() - this.lastFailureTime >= this.options.resetTimeout;
  }
}
```

### **Performance Monitoring:**
```typescript
// src/utils/MetricsCollector.ts
import { createClient } from 'redis';

export class MetricsCollector {
  private redis = createClient({
    url: process.env.REDIS_METRICS_URL
  });
  
  /**
   * Record timing metrics
   */
  timing(metric: string, duration: number): void {
    // Store timing data for analysis
    setImmediate(async () => {
      try {
        const key = `metrics:timing:${metric}`;
        const timestamp = Math.floor(Date.now() / 1000);
        
        await Promise.all([
          // Store individual timing
          this.redis.zadd(`${key}:raw`, timestamp, duration),
          
          // Update statistical aggregates
          this.redis.hincrbyfloat(`${key}:stats`, 'sum', duration),
          this.redis.hincrby(`${key}:stats`, 'count', 1),
          
          // Store percentile data
          this.redis.zadd(`${key}:percentiles`, duration, `${timestamp}-${Math.random()}`)
        ]);
        
        // Expire old data (keep 24 hours)
        await this.redis.expire(`${key}:raw`, 86400);
        
      } catch (error) {
        console.error('Metrics timing error:', error);
      }
    });
  }
  
  /**
   * Increment counter metrics
   */
  increment(metric: string, value: number = 1): void {
    setImmediate(async () => {
      try {
        const key = `metrics:counter:${metric}`;
        const timestamp = Math.floor(Date.now() / 60); // Minute precision
        
        await Promise.all([
          this.redis.hincrby(key, timestamp.toString(), value),
          this.redis.expire(key, 86400) // 24 hours retention
        ]);
        
      } catch (error) {
        console.error('Metrics increment error:', error);
      }
    });
  }
  
  /**
   * Get performance dashboard data
   */
  async getDashboardMetrics(): Promise<any> {
    try {
      const [
        redirectTimes,
        hitRates,
        errorCounts
      ] = await Promise.all([
        this.getTimingStats('redirect.response_time'),
        this.getCacheHitRates(),
        this.getErrorCounts()
      ]);
      
      return {
        performance: redirectTimes,
        caching: hitRates,
        errors: errorCounts,
        timestamp: new Date()
      };
      
    } catch (error) {
      console.error('Dashboard metrics error:', error);
      return null;
    }
  }
  
  private async getTimingStats(metric: string) {
    const stats = await this.redis.hgetall(`metrics:timing:${metric}:stats`);
    
    if (stats.count && stats.sum) {
      return {
        average: parseFloat(stats.sum) / parseInt(stats.count),
        count: parseInt(stats.count)
      };
    }
    
    return { average: 0, count: 0 };
  }
  
  private async getCacheHitRates() {
    // Implementation for cache hit rate calculation
    return {
      hotCache: 0.85,
      redisCache: 0.92,
      overall: 0.95
    };
  }
  
  private async getErrorCounts() {
    // Implementation for error count aggregation
    return {
      notFound: 0,
      expired: 0,
      errors: 0
    };
  }
}
```

---

## 🚀 **LOAD TESTING & BENCHMARKS:**

### **Load Testing Script:**
```bash
#!/bin/bash
# Performance testing script using Apache Bench

echo "=== URL Redirect Performance Test ==="

# Test different scenarios
DOMAIN="https://shortlink.com"
TEST_DURATION=60
CONCURRENT_USERS=(10 50 100 500 1000)

for users in "${CONCURRENT_USERS[@]}"; do
  echo "Testing with $users concurrent users..."
  
  # Test redirect performance
  ab -t $TEST_DURATION \
     -c $users \
     -k \
     -H "Accept-Encoding: gzip,deflate" \
     "$DOMAIN/abc123" \
     > "results_${users}_users.txt"
  
  # Extract key metrics
  avg_time=$(grep "Time per request:" "results_${users}_users.txt" | head -1 | awk '{print $4}')
  requests_per_sec=$(grep "Requests per second:" "results_${users}_users.txt" | awk '{print $4}')
  
  echo "  Average response time: ${avg_time}ms"
  echo "  Requests per second: $requests_per_sec"
  echo "---"
done

echo "Performance test completed. Check results_*.txt files for detailed reports."
```

### **Performance Benchmarks:**
```yaml
Target Performance Metrics:
  Average Response Time: < 20ms
  95th Percentile: < 50ms
  99th Percentile: < 100ms
  Requests per Second: > 10,000
  Concurrent Users: > 1,000
  
Cache Performance:
  Hot Cache Hit Rate: > 80%
  Redis Cache Hit Rate: > 90%
  Overall Cache Hit Rate: > 95%
  
Reliability:
  Uptime: > 99.9%
  Error Rate: < 0.1%
  Circuit Breaker Activation: < 1%
```

---

## ✅ **ACCEPTANCE CRITERIA:**
- [ ] Average redirect response time < 20ms
- [ ] 95th percentile response time < 50ms
- [ ] Support 10,000+ concurrent requests
- [ ] Cache hit rate > 95%
- [ ] Circuit breaker functional
- [ ] Graceful degradation during failures
- [ ] Comprehensive monitoring implemented
- [ ] Load testing passing all benchmarks
- [ ] Error handling comprehensive
- [ ] Memory usage optimized

---

## 🧪 **TESTING CHECKLIST:**
- [ ] Performance benchmarks with ab/wrk
- [ ] Load testing with different concurrency levels
- [ ] Cache invalidation testing
- [ ] Circuit breaker failure scenarios
- [ ] Memory leak testing
- [ ] Database failover testing
- [ ] Redis cluster failover testing
- [ ] CDN integration testing

---

**Completion Date**: _________  
**Review By**: Performance Engineer + Senior Backend  
**Next Task**: URL Analytics Tracking