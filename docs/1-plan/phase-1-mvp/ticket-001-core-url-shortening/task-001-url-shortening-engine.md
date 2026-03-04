# TASK: URL Shortening Core Engine

**Ticket**: Core URL Shortening  
**Priority**: P0 (CRITICAL - HIGHEST PRIORITY)  
**Assignee**: Senior Backend Developer  
**Estimate**: 4 days  
**Dependencies**: Database Setup  

## 📋 TASK OVERVIEW

**Objective**: Implement core URL shortening algorithm និង API endpoints  
**Success Criteria**: Users can create និង redirect short URLs successfully  

---

## 🎯 **CORE FUNCTIONALITY REQUIREMENTS:**

### **URL Shortening Algorithm:**
- [ ] Generate unique short codes (6-8 characters)
- [ ] Base62 encoding for URL-safe characters 
- [ ] Collision detection និង resolution
- [ ] Custom alias support (optional)
- [ ] Domain validation និង sanitization

### **Core API Endpoints:**
- [ ] `POST /api/shorten` - Create short URL
- [ ] `GET /{shortCode}` - Redirect to original URL  
- [ ] `GET /api/urls/{shortCode}` - Get URL details
- [ ] `PUT /api/urls/{shortCode}` - Update URL
- [ ] `DELETE /api/urls/{shortCode}` - Delete URL

### **Performance Requirements:**
- [ ] < 50ms response time for redirects
- [ ] Support 10,000+ concurrent requests
- [ ] 99.9% uptime SLA
- [ ] Database query optimization

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **📊 DATABASE SCHEMA ANALYSIS:**

#### **Primary URL Storage Table:**
```sql
-- PostgreSQL Schema for URL Storage
CREATE TABLE short_urls (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(50) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    user_id BIGINT REFERENCES users(id),
    custom_alias VARCHAR(50) UNIQUE,
    
    -- URL Metadata
    title VARCHAR(500),
    description TEXT,
    domain_name VARCHAR(255),
    url_hash VARCHAR(64), -- SHA-256 của original_url để deduplication
    
    -- Security & Access Control
    is_active BOOLEAN DEFAULT TRUE,
    password_protected BOOLEAN DEFAULT FALSE,
    password_hash VARCHAR(255),
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Performance Counters (Denormalized)
    total_clicks BIGINT DEFAULT 0,
    unique_clicks BIGINT DEFAULT 0,
    last_clicked_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit Trail
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_ip INET,
    
    -- Constraints
    CONSTRAINT valid_short_code CHECK (short_code ~ '^[a-zA-Z0-9]{4,50}$'),
    CONSTRAINT valid_url CHECK (original_url ~ '^https?://.*'),
    CONSTRAINT future_expiry CHECK (expires_at IS NULL OR expires_at > created_at)
);

-- Performance Indexes
CREATE UNIQUE INDEX idx_short_urls_code ON short_urls(short_code);
CREATE INDEX idx_short_urls_user ON short_urls(user_id, created_at DESC);
CREATE INDEX idx_short_urls_active ON short_urls(is_active, expires_at);
CREATE INDEX idx_short_urls_hash ON short_urls(url_hash) WHERE url_hash IS NOT NULL;
CREATE INDEX idx_short_urls_domain ON short_urls(domain_name);
CREATE INDEX idx_short_urls_clicks ON short_urls(total_clicks DESC);

-- Partial index for active URLs only (performance boost)
CREATE INDEX idx_short_urls_lookup ON short_urls(short_code, is_active) 
    WHERE is_active = TRUE;
```

#### **Click Analytics Table (Hot Data):**
```sql
-- Separate table for high-frequency click data
CREATE TABLE url_clicks (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(50) NOT NULL,
    
    -- Request Information
    ip_address INET NOT NULL,
    user_agent TEXT,
    referer TEXT,
    
    -- Geographic Data
    country_code CHAR(2),
    city VARCHAR(100),
    
    -- Analytics
    is_unique_visitor BOOLEAN DEFAULT TRUE,
    session_id VARCHAR(100),
    
    -- Timestamp
    clicked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Foreign Key
    FOREIGN KEY (short_code) REFERENCES short_urls(short_code) ON DELETE CASCADE
);

-- Time-based partitioning for analytics
CREATE INDEX idx_clicks_time ON url_clicks(clicked_at DESC);
CREATE INDEX idx_clicks_code ON url_clicks(short_code, clicked_at DESC);
CREATE INDEX idx_clicks_unique ON url_clicks(short_code, ip_address, clicked_at);
```

---

### **🔢 BASE62 ENCODING ALGORITHM:**

#### **Character Set Analysis:**
```typescript
// Base62 Character Set: 62 ký tự
const BASE62_ALPHABET = '0123456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz';
// Loại bỏ: I, L, O (confusion với 1, l, 0)
// Total: 10 số + 26 chữ hoa + 26 chữ thường = 62 ký tự

// Với 6 ký tự: 62^6 = 56,800,235,584 possible combinations
// Với 7 ký tự: 62^7 = 3,521,614,606,208 combinations
```

#### **Thuật toán Encoding/Decoding:**
```typescript
// src/utils/Base62Encoder.ts
export class Base62Encoder {
    private static readonly ALPHABET = '0123456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz';
    private static readonly BASE = 62;
    
    /**
     * Encode số thành Base62 string
     * @param num - Số cần encode (từ auto-increment ID)
     * @param minLength - Độ dài tối thiểu (default: 6)
     */
    static encode(num: number, minLength: number = 6): string {
        if (num === 0) return this.ALPHABET[0].repeat(minLength);
        
        let result = '';
        let n = num;
        
        while (n > 0) {
            result = this.ALPHABET[n % this.BASE] + result;
            n = Math.floor(n / this.BASE);
        }
        
        // Pad với ký tự đầu tiên để đạt minLength
        while (result.length < minLength) {
            result = this.ALPHABET[0] + result;
        }
        
        return result;
    }
    
    /**
     * Decode Base62 string thành số
     */
    static decode(str: string): number {
        let result = 0;
        const len = str.length;
        
        for (let i = 0; i < len; i++) {
            const char = str[i];
            const value = this.ALPHABET.indexOf(char);
            
            if (value === -1) {
                throw new Error(`Invalid character '${char}' in Base62 string`);
            }
            
            result = result * this.BASE + value;
        }
        
        return result;
    }
    
    /**
     * Generate random Base62 string
     * Dùng cho custom alias hoặc fallback
     */
    static generateRandom(length: number = 6): string {
        let result = '';
        
        for (let i = 0; i < length; i++) {
            const randomIndex = Math.floor(Math.random() * this.BASE);
            result += this.ALPHABET[randomIndex];
        }
        
        return result;
    }
    
    /**
     * Validate Base62 string format
     */
    static isValid(str: string): boolean {
        if (!str || str.length < 4 || str.length > 50) {
            return false;
        }
        
        return str.split('').every(char => this.ALPHABET.includes(char));
    }
}
```

#### **ID-Based vs Random Generation Strategy:**
```typescript
// src/utils/ShortCodeGenerator.ts
export class ShortCodeGenerator {
    
    /**
     * Strategy 1: Counter-based (Deterministic)
     * Pros: No collision, predictable, sequential
     * Cons: Guessable, reveals volume
     */
    static async generateFromCounter(): Promise<string> {
        // Lấy next sequence value từ database
        const result = await db.query('SELECT nextval(\'short_url_seq\') as id');
        const id = result.rows[0].id;
        
        // Encode ID thành Base62
        return Base62Encoder.encode(id, 6);
    }
    
    /**
     * Strategy 2: Random with collision check (Secure)
     * Pros: Unpredictable, secure
     * Cons: Potential collisions, requires checking
     */
    static async generateRandom(attempts: number = 5): Promise<string> {
        for (let i = 0; i < attempts; i++) {
            const candidate = Base62Encoder.generateRandom(6);
            
            // Check collision trong database
            const exists = await this.checkCodeExists(candidate);
            
            if (!exists) {
                return candidate;
            }
            
            // Increase length after failed attempts
            if (i === attempts - 1) {
                return Base62Encoder.generateRandom(7); // Fallback to 7 chars
            }
        }
        
        throw new Error('Failed to generate unique short code');
    }
    
    /**
     * Strategy 3: Hybrid approach (Recommended)
     * Counter-based với random offset để obfuscate
     */
    static async generateHybrid(): Promise<string> {
        const counter = await this.getNextCounter();
        const randomOffset = Math.floor(Math.random() * 10000);
        const obfuscatedId = counter * 10000 + randomOffset;
        
        return Base62Encoder.encode(obfuscatedId, 6);
    }
    
    private static async checkCodeExists(shortCode: string): Promise<boolean> {
        const result = await db.query(
            'SELECT 1 FROM short_urls WHERE short_code = $1 LIMIT 1',
            [shortCode]
        );
        return result.rowCount > 0;
    }
    
    private static async getNextCounter(): Promise<number> {
        const result = await db.query('SELECT nextval(\'short_url_counter\') as id');
        return result.rows[0].id;
    }
}
```

---

### **🔍 LOOKUP & EXISTENCE CHECK MECHANISM:**

#### **Multi-tier Lookup Strategy:**
```typescript
// src/services/LookupService.ts
export class LookupService {
    private redis: Redis;
    private postgres: Pool;
    
    constructor() {
        this.redis = new Redis(process.env.REDIS_URL);
        this.postgres = new Pool({
            connectionString: process.env.DATABASE_URL,
            max: 20 // Connection pool size
        });
    }
    
    /**
     * Tier 1: Redis Cache Lookup (< 1ms)
     * Tier 2: Database Query (< 10ms)
     * Tier 3: 404 Cache (prevent repeated lookups)
     */
    async findShortCode(shortCode: string): Promise<URLRecord | null> {
        // Validate input format first
        if (!Base62Encoder.isValid(shortCode)) {
            return null;
        }
        
        // Tier 1: Hot cache lookup
        const cached = await this.getFromCache(shortCode);
        if (cached !== undefined) {
            return cached; // null means "not found" is cached
        }
        
        // Tier 2: Database lookup
        const dbResult = await this.getFromDatabase(shortCode);
        
        // Cache result (both positive and negative)
        await this.cacheResult(shortCode, dbResult);
        
        return dbResult;
    }
    
    /**
     * Redis Cache Layer với TTL strategy
     */
    private async getFromCache(shortCode: string): Promise<URLRecord | null | undefined> {
        try {
            const cacheKey = `url:${shortCode}`;
            const cached = await this.redis.get(cacheKey);
            
            if (cached === null) {
                return undefined; // Cache miss
            }
            
            if (cached === 'NOT_FOUND') {
                return null; // Negative cache hit
            }
            
            return JSON.parse(cached) as URLRecord;
            
        } catch (error) {
            console.error('Cache lookup error:', error);
            return undefined; // Fallback to database
        }
    }
    
    /**
     * Database lookup với optimized query
     */
    private async getFromDatabase(shortCode: string): Promise<URLRecord | null> {
        const query = `
            SELECT 
                short_code,
                original_url,
                is_active,
                expires_at,
                password_protected,
                total_clicks
            FROM short_urls 
            WHERE short_code = $1 
                AND is_active = TRUE 
                AND (expires_at IS NULL OR expires_at > NOW())
            LIMIT 1
        `;
        
        try {
            const result = await this.postgres.query(query, [shortCode]);
            
            if (result.rowCount === 0) {
                return null;
            }
            
            return result.rows[0] as URLRecord;
            
        } catch (error) {
            console.error('Database lookup error:', error);
            throw new Error('Database lookup failed');
        }
    }
    
    /**
     * Cache strategy với different TTL
     */
    private async cacheResult(shortCode: string, result: URLRecord | null): Promise<void> {
        const cacheKey = `url:${shortCode}`;
        
        try {
            if (result === null) {
                // Negative caching - ngăn repeated 404 lookups
                await this.redis.setex(cacheKey, 300, 'NOT_FOUND'); // 5 minutes
            } else {
                // Positive caching - cache URL data
                const ttl = this.calculateTTL(result);
                await this.redis.setex(cacheKey, ttl, JSON.stringify(result));
            }
        } catch (error) {
            console.error('Cache write error:', error);
            // Non-blocking error
        }
    }
    
    /**
     * Dynamic TTL based on URL characteristics
     */
    private calculateTTL(url: URLRecord): number {
        // Popular URLs (high clicks) = longer cache
        if (url.total_clicks > 10000) {
            return 7200; // 2 hours
        }
        
        // Recent URLs = shorter cache (may be updated)
        const ageInHours = (Date.now() - new Date(url.created_at).getTime()) / (1000 * 60 * 60);
        if (ageInHours < 24) {
            return 1800; // 30 minutes
        }
        
        // Default cache time
        return 3600; // 1 hour
    }
    
    /**
     * Batch existence check for bulk operations
     */
    async checkMultipleExists(shortCodes: string[]): Promise<Record<string, boolean>> {
        if (shortCodes.length === 0) return {};
        
        // Build parameterized query
        const placeholders = shortCodes.map((_, index) => `$${index + 1}`).join(',');
        const query = `
            SELECT short_code 
            FROM short_urls 
            WHERE short_code IN (${placeholders}) 
                AND is_active = TRUE
        `;
        
        const result = await this.postgres.query(query, shortCodes);
        const existingCodes = new Set(result.rows.map(row => row.short_code));
        
        // Build result map
        const results: Record<string, boolean> = {};
        for (const code of shortCodes) {
            results[code] = existingCodes.has(code);
        }
        
        return results;
    }
}
```

---

### **Core URL Model (Node.js + TypeScript):**
```typescript
// src/models/ShortURL.ts
import { Document, Schema, model } from 'mongoose';
import { nanoid, customAlphabet } from 'nanoid';

// URL-safe alphabet (excluding ambiguous characters)
const alphabet = '0123456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz';
const generateShortCode = customAlphabet(alphabet, 6);

export interface IShortURL extends Document {
  shortCode: string;
  originalUrl: string;
  userId?: string;
  customAlias?: string;
  title?: string;
  description?: string;
  
  // Settings
  isActive: boolean;
  passwordProtected: boolean;
  passwordHash?: string;
  expiresAt?: Date;
  
  // Statistics
  totalClicks: number;
  uniqueClicks: number;
  lastClickedAt?: Date;
  
  // Metadata
  createdAt: Date;
  updatedAt: Date;
}

const ShortURLSchema = new Schema<IShortURL>({
  shortCode: {
    type: String,
    required: true,
    unique: true,
    index: true,
    minlength: 4,
    maxlength: 50
  },
  originalUrl: {
    type: String,
    required: true,
    validate: {
      validator: function(url: string) {
        // URL validation regex
        const urlRegex = /^https?:\/\/.+\..+/;
        return urlRegex.test(url);
      },
      message: 'Invalid URL format'
    }
  },
  userId: {
    type: Schema.Types.ObjectId,
    ref: 'User',
    index: true
  },
  customAlias: {
    type: String,
    unique: true,
    sparse: true,
    minlength: 3,
    maxlength: 50,
    match: /^[a-zA-Z0-9_-]+$/
  },
  title: {
    type: String,
    maxlength: 500,
    trim: true
  },
  description: {
    type: String,
    maxlength: 2000,
    trim: true
  },
  
  // Settings
  isActive: {
    type: Boolean,
    default: true,
    index: true
  },
  passwordProtected: {
    type: Boolean,
    default: false
  },
  passwordHash: String,
  expiresAt: {
    type: Date,
    index: true
  },
  
  // Statistics (denormalized for performance)
  totalClicks: {
    type: Number,
    default: 0,
    min: 0
  },
  uniqueClicks: {
    type: Number,
    default: 0,
    min: 0
  },
  lastClickedAt: Date,
  
}, {
  timestamps: true,
  collection: 'short_urls'
});

// Compound indexes for performance
ShortURLSchema.index({ userId: 1, createdAt: -1 });
ShortURLSchema.index({ isActive: 1, expiresAt: 1 });
ShortURLSchema.index({ shortCode: 1, isActive: 1 });

// Pre-save middleware for short code generation
ShortURLSchema.pre<IShortURL>('save', async function(next) {
  if (this.isNew && !this.shortCode) {
    // Strategy: Hybrid approach (counter + random)
    try {
      this.shortCode = await ShortCodeGenerator.generateHybrid();
    } catch (error) {
      // Fallback to pure random
      let attempts = 0;
      const maxAttempts = 5;
      
      while (attempts < maxAttempts) {
        const candidate = Base62Encoder.generateRandom(6 + attempts); // Increase length on retry
        
        // Optimized existence check
        const existing = await model('ShortURL').findOne(
          { shortCode: candidate },
          { _id: 1 } // Only fetch ID for performance
        ).lean();
        
        if (!existing) {
          this.shortCode = candidate;
          break;
        }
        
        attempts++;
      }
      
      if (!this.shortCode) {
        throw new Error('Unable to generate unique short code after maximum attempts');
      }
    }
  }
  
  // Generate URL hash for deduplication
  if (this.isModified('originalUrl') || this.isNew) {
    const crypto = await import('crypto');
    this.set('urlHash', crypto.createHash('sha256').update(this.originalUrl).digest('hex'));
  }
  
  next();
});

// Add URL hash field to schema
ShortURLSchema.add({
  urlHash: {
    type: String,
    index: true,
    length: 64
  }
});

export const ShortURL = model<IShortURL>('ShortURL', ShortURLSchema);
```

### **URL Shortening Service:**
```typescript
// src/services/URLShorteningService.ts
import { ShortURL, IShortURL } from '../models/ShortURL';
import { URLValidator } from '../utils/URLValidator';
import { Cache } from '../utils/Cache';
import { EventEmitter } from 'events';

export class URLShorteningService extends EventEmitter {
  private cache = new Cache();
  
  /**
   * Create a new short URL
   */
  async createShortURL(data: {
    originalUrl: string;
    userId?: string;
    customAlias?: string;
    title?: string;
    description?: string;
    expiresAt?: Date;
    password?: string;
  }): Promise<IShortURL> {
    
    // 1. Validate original URL
    const validatedUrl = await URLValidator.validateAndNormalize(data.originalUrl);
    
    // 2. Check for existing URL (deduplication)
    if (data.userId) {
      const existing = await ShortURL.findOne({
        originalUrl: validatedUrl,
        userId: data.userId,
        isActive: true
      });
      
      if (existing) {
        return existing; // Return existing instead of creating duplicate
      }
    }
    
    // 3. Handle custom alias
    if (data.customAlias) {
      const aliasExists = await ShortURL.findOne({ 
        shortCode: data.customAlias 
      });
      
      if (aliasExists) {
        throw new Error('Custom alias already taken');
      }
    }
    
    // 4. Create short URL
    const shortURL = new ShortURL({
      originalUrl: validatedUrl,
      userId: data.userId,
      shortCode: data.customAlias, // Will auto-generate if empty
      title: data.title,
      description: data.description,
      expiresAt: data.expiresAt,
      passwordProtected: !!data.password,
      passwordHash: data.password ? await this.hashPassword(data.password) : undefined
    });
    
    await shortURL.save();
    
    // 5. Cache the URL for fast redirects
    await this.cache.set(`url:${shortURL.shortCode}`, {
      originalUrl: shortURL.originalUrl,
      isActive: shortURL.isActive,
      expiresAt: shortURL.expiresAt,
      passwordProtected: shortURL.passwordProtected
    }, 3600); // 1 hour cache
    
    // 6. Emit event for analytics
    this.emit('url_created', {
      shortCode: shortURL.shortCode,
      userId: data.userId,
      originalUrl: validatedUrl
    });
    
    return shortURL;
  }
  
  /**
   * Get redirect URL and track click
   */
  async getRedirectUrl(shortCode: string, requestData: {
    ipAddress: string;
    userAgent: string;
    referer?: string;
    password?: string;
  }): Promise<{ redirectUrl: string; shouldTrack: boolean }> {
    
    // 1. Validate short code format first
    if (!Base62Encoder.isValid(shortCode)) {
      throw new Error('Invalid short code format');
    }
    
    // 2. Multi-tier lookup strategy
    const lookupService = new LookupService();
    const urlRecord = await lookupService.findShortCode(shortCode);
    
    if (!urlRecord) {
      throw new Error('URL not found');
    }
    
    // 3. Check expiration
    if (urlRecord.expires_at && new Date(urlRecord.expires_at) < new Date()) {
      throw new Error('URL has expired');
    }
    
    const urlData = {
      originalUrl: urlRecord.original_url,
      isActive: urlRecord.is_active,
      expiresAt: urlRecord.expires_at,
      passwordProtected: urlRecord.password_protected
    };
    
    // 4. Check password protection
    if (urlData.passwordProtected && !requestData.password) {
      throw new Error('Password required');
    }
    
    if (urlData.passwordProtected && requestData.password) {
      const shortURL = await ShortURL.findOne({ shortCode });
      const isValidPassword = await this.verifyPassword(
        requestData.password, 
        shortURL?.passwordHash || ''
      );
      
      if (!isValidPassword) {
        throw new Error('Invalid password');
      }
    }
    
    // 5. Emit tracking event (async)
    setImmediate(() => {
      this.emit('url_clicked', {
        shortCode,
        ipAddress: requestData.ipAddress,
        userAgent: requestData.userAgent,
        referer: requestData.referer,
        timestamp: new Date()
      });
    });
    
    return {
      redirectUrl: urlData.originalUrl,
      shouldTrack: true
    };
  }
  
  /**
   * Get URL statistics and details
   */
  async getURLDetails(shortCode: string, userId?: string): Promise<IShortURL> {
    const query: any = { shortCode, isActive: true };
    
    // Only return URLs owned by user (privacy)
    if (userId) {
      query.userId = userId;
    }
    
    const shortURL = await ShortURL.findOne(query);
    
    if (!shortURL) {
      throw new Error('URL not found or access denied');
    }
    
    return shortURL;
  }
  
  /**
   * Update existing short URL
   */
  async updateShortURL(
    shortCode: string, 
    userId: string, 
    updateData: Partial<IShortURL>
  ): Promise<IShortURL> {
    
    const shortURL = await ShortURL.findOne({
      shortCode,
      userId,
      isActive: true
    });
    
    if (!shortURL) {
      throw new Error('URL not found or access denied');
    }
    
    // Validate updated URL if provided
    if (updateData.originalUrl) {
      updateData.originalUrl = await URLValidator.validateAndNormalize(
        updateData.originalUrl
      );
    }
    
    // Update fields
    Object.assign(shortURL, updateData);
    shortURL.updatedAt = new Date();
    
    await shortURL.save();
    
    // Invalidate cache
    await this.cache.delete(`url:${shortCode}`);
    
    // Emit update event
    this.emit('url_updated', {
      shortCode,
      userId,
      updateData
    });
    
    return shortURL;
  }
  
  /**
   * Soft delete URL
   */
  async deleteShortURL(shortCode: string, userId: string): Promise<boolean> {
    const result = await ShortURL.updateOne(
      { shortCode, userId },
      { isActive: false, updatedAt: new Date() }
    );
    
    if (result.modifiedCount === 0) {
      throw new Error('URL not found or access denied');
    }
    
    // Remove from cache
    await this.cache.delete(`url:${shortCode}`);
    
    // Emit deletion event
    this.emit('url_deleted', {
      shortCode,
      userId
    });
    
    return true;
  }
  
  private async hashPassword(password: string): Promise<string> {
    const bcrypt = await import('bcrypt');
    return bcrypt.hash(password, 12);
  }
  
  private async verifyPassword(password: string, hash: string): Promise<boolean> {
    const bcrypt = await import('bcrypt');
    return bcrypt.compare(password, hash);
  }
}
```

### **URL Shortening API Controller:**
```typescript
// src/controllers/URLController.ts
import { Request, Response, NextFunction } from 'express';
import { URLShorteningService } from '../services/URLShorteningService';
import { RateLimiter } from '../utils/RateLimiter';
import { getClientIP } from '../utils/ClientUtils';

export class URLController {
  private urlService = new URLShorteningService();
  private rateLimiter = new RateLimiter();
  
  /**
   * POST /api/shorten
   * Create a new short URL
   */
  async createShortURL(req: Request, res: Response, next: NextFunction) {
    try {
      // Rate limiting
      const clientIP = getClientIP(req);
      await this.rateLimiter.checkLimit(clientIP, 'create_url', 100, 3600); // 100 per hour
      
      const {
        url,
        customAlias,
        title,
        description,
        expiresAt,
        password
      } = req.body;
      
      // Input validation
      if (!url || typeof url !== 'string') {
        return res.status(400).json({
          success: false,
          error: 'URL is required'
        });
      }
      
      if (customAlias && !/^[a-zA-Z0-9_-]{3,50}$/.test(customAlias)) {
        return res.status(400).json({
          success: false,
          error: 'Custom alias must be 3-50 characters (alphanumeric, underscore, hyphen only)'
        });
      }
      
      const shortURL = await this.urlService.createShortURL({
        originalUrl: url,
        userId: req.user?.id,
        customAlias,
        title,
        description,
        expiresAt: expiresAt ? new Date(expiresAt) : undefined,
        password
      });
      
      res.status(201).json({
        success: true,
        data: {
          shortCode: shortURL.shortCode,
          shortUrl: `${process.env.BASE_URL}/${shortURL.shortCode}`,
          originalUrl: shortURL.originalUrl,
          title: shortURL.title,
          createdAt: shortURL.createdAt,
          expiresAt: shortURL.expiresAt
        }
      });
      
    } catch (error) {
      next(error);
    }
  }
  
  /**
   * GET /{shortCode}
   * Redirect to original URL
   */
  async redirectToOriginal(req: Request, res: Response, next: NextFunction) {
    try {
      const { shortCode } = req.params;
      const { password } = req.query;
      
      if (!shortCode || !/^[a-zA-Z0-9_-]{3,50}$/.test(shortCode)) {
        return res.status(404).json({
          success: false,
          error: 'Invalid short code'
        });
      }
      
      const result = await this.urlService.getRedirectUrl(shortCode, {
        ipAddress: getClientIP(req),
        userAgent: req.get('User-Agent') || '',
        referer: req.get('Referer'),
        password: password as string
      });
      
      // Permanent redirect for SEO
      res.redirect(301, result.redirectUrl);
      
    } catch (error) {
      if (error.message === 'URL not found') {
        return res.status(404).json({
          success: false,
          error: 'Short URL not found'
        });
      }
      
      if (error.message === 'Password required') {
        return res.status(401).json({
          success: false,
          error: 'Password required',
          requirePassword: true
        });
      }
      
      if (error.message === 'Invalid password') {
        return res.status(403).json({
          success: false,
          error: 'Invalid password'
        });
      }
      
      if (error.message === 'URL has expired') {
        return res.status(410).json({
          success: false,
          error: 'This link has expired'
        });
      }
      
      next(error);
    }
  }
  
  /**
   * GET /api/urls/{shortCode}
   * Get URL details and statistics
   */
  async getURLDetails(req: Request, res: Response, next: NextFunction) {
    try {
      const { shortCode } = req.params;
      
      const urlDetails = await this.urlService.getURLDetails(
        shortCode,
        req.user?.id
      );
      
      res.json({
        success: true,
        data: {
          shortCode: urlDetails.shortCode,
          originalUrl: urlDetails.originalUrl,
          title: urlDetails.title,
          description: urlDetails.description,
          totalClicks: urlDetails.totalClicks,
          uniqueClicks: urlDetails.uniqueClicks,
          lastClickedAt: urlDetails.lastClickedAt,
          createdAt: urlDetails.createdAt,
          expiresAt: urlDetails.expiresAt,
          isActive: urlDetails.isActive
        }
      });
      
    } catch (error) {
      if (error.message === 'URL not found or access denied') {
        return res.status(404).json({
          success: false,
          error: 'URL not found'
        });
      }
      
      next(error);
    }
  }
}
```

---

## � **PERFORMANCE & COLLISION ANALYSIS:**

### **Collision Probability Mathematics:**
```typescript
// Base62 Collision Analysis
interface CollisionStats {
    codeLength: number;
    totalPossible: bigint;
    safeCapacity: bigint; // 50% capacity để tránh collision cao
    collisionAt50: number; // Số URLs khi collision = 50%
}

const COLLISION_ANALYSIS: CollisionStats[] = [
    {
        codeLength: 4,
        totalPossible: BigInt(62 ** 4), // 14,776,336
        safeCapacity: BigInt(62 ** 4 / 2), // 7,388,168
        collisionAt50: Math.sqrt(62 ** 4) // ~3,844 URLs (Birthday Paradox)
    },
    {
        codeLength: 5,
        totalPossible: BigInt(62 ** 5), // 916,132,832
        safeCapacity: BigInt(62 ** 5 / 2), // 458,066,416
        collisionAt50: Math.sqrt(62 ** 5) // ~30,268 URLs
    },
    {
        codeLength: 6,
        totalPossible: BigInt(62 ** 6), // 56,800,235,584
        safeCapacity: BigInt(62 ** 6 / 2), // 28,400,117,792
        collisionAt50: Math.sqrt(62 ** 6) // ~238,328 URLs
    },
    {
        codeLength: 7,
        totalPossible: BigInt(62 ** 7), // 3,521,614,606,208
        safeCapacity: BigInt(62 ** 7 / 2), // 1,760,807,303,104
        collisionAt50: Math.sqrt(62 ** 7) // ~1,876,835 URLs
    }
];

/**
 * Tính xác suất collision dựa trên số URLs hiện tại
 */
function calculateCollisionProbability(urlCount: number, codeLength: number): number {
    const totalPossible = 62 ** codeLength;
    
    // Birthday Paradox Formula: P(collision) ≈ 1 - e^(-n²/2N)
    const exponent = -(urlCount ** 2) / (2 * totalPossible);
    return 1 - Math.exp(exponent);
}

/**
 * Recommend optimal code length based on expected volume
 */
function recommendCodeLength(expectedVolume: number): number {
    for (const stats of COLLISION_ANALYSIS) {
        if (expectedVolume < Number(stats.safeCapacity)) {
            return stats.codeLength;
        }
    }
    return 8; // Fallback for very high volume
}
```

### **Advanced Collision Handling:**
```typescript
// src/utils/CollisionHandler.ts
export class CollisionHandler {
    private static readonly MAX_RETRY_ATTEMPTS = 3;
    private static readonly INITIAL_CODE_LENGTH = 6;
    
    /**
     * Intelligent collision resolution với escalating strategies
     */
    static async resolveCollision(attempt: number): Promise<string> {
        const strategies = [
            () => this.randomRetry(6),           // Strategy 1: Random 6-char
            () => this.randomRetry(7),           // Strategy 2: Longer random
            () => this.timestampBased(),         // Strategy 3: Timestamp-based
            () => this.counterWithSalt()         // Strategy 4: Counter + salt
        ];
        
        if (attempt >= strategies.length) {
            throw new Error(`Maximum collision resolution attempts exceeded`);
        }
        
        return strategies[attempt]();
    }
    
    /**
     * Strategy 1 & 2: Pure random với variable length
     */
    private static randomRetry(length: number): string {
        return Base62Encoder.generateRandom(length);
    }
    
    /**
     * Strategy 3: Timestamp-based generation
     * Pros: Temporal ordering, very low collision
     * Cons: Predictable pattern
     */
    private static timestampBased(): string {
        const timestamp = Date.now();
        const randomSuffix = Math.floor(Math.random() * 1000);
        const combined = timestamp * 1000 + randomSuffix;
        
        return Base62Encoder.encode(combined).slice(-8); // Last 8 chars
    }
    
    /**
     * Strategy 4: Database counter với random salt
     * Pros: Guaranteed unique, performant
     * Cons: Requires database round-trip
     */
    private static async counterWithSalt(): Promise<string> {
        const counter = await this.getGlobalCounter();
        const salt = Math.floor(Math.random() * 62 ** 3); // 3-digit salt
        const combined = counter * (62 ** 3) + salt;
        
        return Base62Encoder.encode(combined, 8);
    }
    
    private static async getGlobalCounter(): Promise<number> {
        // Atomic increment in Redis for performance
        const redis = new Redis(process.env.REDIS_URL);
        return redis.incr('shortlink:global_counter');
    }
}
```

---

### **🚀 PERFORMANCE OPTIMIZATION STRATEGIES:**

#### **Database Optimization:**
```sql
-- 1. Optimized existence check query
EXPLAIN ANALYZE
SELECT 1 FROM short_urls 
WHERE short_code = 'abc123' 
    AND is_active = true 
LIMIT 1;

-- Expected: Index Scan using idx_short_urls_lookup (cost=0.43..8.45)

-- 2. Batch existence check for bulk operations
WITH codes_to_check(code) AS (
    VALUES ('abc123'), ('def456'), ('ghi789')
)
SELECT 
    c.code,
    EXISTS(
        SELECT 1 FROM short_urls s 
        WHERE s.short_code = c.code AND s.is_active = true
    ) as exists
FROM codes_to_check c;

-- 3. Statistics query for collision monitoring
SELECT 
    COUNT(*) as total_urls,
    MIN(LENGTH(short_code)) as min_length,
    MAX(LENGTH(short_code)) as max_length,
    AVG(LENGTH(short_code)) as avg_length,
    COUNT(DISTINCT LENGTH(short_code)) as length_varieties
FROM short_urls 
WHERE is_active = true;
```

#### **Redis Caching Strategy:**
```typescript
// src/utils/AdvancedCache.ts
export class AdvancedCache {
    private redis: Redis;
    
    constructor() {
        this.redis = new Redis({
            host: process.env.REDIS_HOST,
            port: 6379,
            // Connection pooling
            maxRetriesPerRequest: 3,
            retryDelayOnFailover: 100,
            enableReadyCheck: true,
            // Performance tuning
            lazyConnect: true,
            keepAlive: 30000,
            family: 4
        });
    }
    
    /**
     * Multi-level caching với different TTLs
     */
    async cacheURL(shortCode: string, data: URLRecord): Promise<void> {
        const pipeline = this.redis.pipeline();
        
        // Level 1: Hot cache (high-access URLs)
        pipeline.setex(`hot:${shortCode}`, 7200, JSON.stringify(data)); // 2 hours
        
        // Level 2: Standard cache
        pipeline.setex(`url:${shortCode}`, 3600, JSON.stringify(data)); // 1 hour
        
        // Level 3: Existence cache (just boolean)
        pipeline.setex(`exists:${shortCode}`, 1800, '1'); // 30 minutes
        
        await pipeline.exec();
    }
    
    /**
     * Intelligent cache lookup với fallback layers
     */
    async getURL(shortCode: string): Promise<URLRecord | null | undefined> {
        // Try hot cache first
        let cached = await this.redis.get(`hot:${shortCode}`);
        if (cached) {
            // Promote to hot cache if accessed
            return JSON.parse(cached);
        }
        
        // Try standard cache
        cached = await this.redis.get(`url:${shortCode}`);
        if (cached) {
            const data = JSON.parse(cached);
            // Async promote to hot cache
            setImmediate(() => this.promoteToHot(shortCode, data));
            return data;
        }
        
        // Try existence cache (avoid DB query for 404s)
        const exists = await this.redis.get(`exists:${shortCode}`);
        if (exists === '0') {
            return null; // Confirmed non-existent
        }
        
        return undefined; // Cache miss, need DB query
    }
    
    /**
     * Negative caching for 404 prevention
     */
    async cacheNonExistent(shortCode: string): Promise<void> {
        const pipeline = this.redis.pipeline();
        
        // Cache negative result
        pipeline.setex(`exists:${shortCode}`, 300, '0'); // 5 minutes
        pipeline.setex(`url:${shortCode}`, 300, 'NOT_FOUND');
        
        await pipeline.exec();
    }
    
    private async promoteToHot(shortCode: string, data: URLRecord): Promise<void> {
        try {
            await this.redis.setex(`hot:${shortCode}`, 7200, JSON.stringify(data));
        } catch (error) {
            // Non-blocking error
            console.error('Failed to promote to hot cache:', error);
        }
    }
}
```

---

### **📈 ALGORITHM COMPARISON:**

| **Strategy** | **Pros** | **Cons** | **Use Case** | **Performance** |
|-------------|----------|----------|--------------|----------------|
| **Counter-based** | • No collisions<br>• Predictable<br>• Sequential | • Guessable<br>• Reveals volume<br>• Single point failure | Internal tools,<br>Admin URLs | ⭐⭐⭐⭐⭐ |
| **Pure Random** | • Unpredictable<br>• Secure<br>• Distributed | • Collision risk<br>• Retry overhead<br>• No ordering | Public URLs,<br>Security-critical | ⭐⭐⭐ |
| **Hybrid** | • Balanced security<br>• Low collision<br>• Scalable | • Moderate complexity<br>• Still somewhat predictable | Production apps,<br>High volume | ⭐⭐⭐⭐ |
| **Timestamp-based** | • Temporal ordering<br>• Very low collision<br>• No DB dependency | • Predictable<br>• Time-based attacks<br>• Clock sync issues | Logging,<br>Analytics URLs | ⭐⭐⭐⭐ |

### **Recommended Implementation Strategy:**
```typescript
// Production-ready generator với fallbacks
export class ProductionShortCodeGenerator {
    
    /**
     * Primary generation method với intelligent fallbacks
     */
    static async generate(): Promise<string> {
        // Step 1: Try hybrid approach (recommended)
        try {
            return await this.generateHybrid();
        } catch (error) {
            console.warn('Hybrid generation failed, falling back to random:', error);
        }
        
        // Step 2: Fallback to collision-resistant random
        try {
            return await this.generateCollisionResistant();
        } catch (error) {
            console.warn('Random generation failed, falling back to timestamp:', error);
        }
        
        // Step 3: Ultimate fallback to timestamp-based
        return this.generateTimestampBased();
    }
    
    private static async generateCollisionResistant(): Promise<string> {
        const maxAttempts = 5;
        
        for (let attempt = 0; attempt < maxAttempts; attempt++) {
            const code = await CollisionHandler.resolveCollision(attempt);
            
            // Quick existence check
            const exists = await this.fastExistenceCheck(code);
            if (!exists) {
                return code;
            }
        }
        
        throw new Error('Failed to generate unique code after maximum attempts');
    }
    
    **private static async fastExistenceCheck(code: string): Promise<boolean> {
        // Use Redis for ultra-fast existence check
        const redis = new Redis(process.env.REDIS_URL);
        
        // Check cache first
        const cached = await redis.get(`exists:${code}`);
        if (cached === '1') return true;
        if (cached === '0') return false;
        
        // Fallback to database
        const result = await db.query(
            'SELECT 1 FROM short_urls WHERE short_code = $1 LIMIT 1',
            [code]
        );
        
        const exists = result.rowCount > 0;
        
        // Cache result
        await redis.setex(`exists:${code}`, 1800, exists ? '1' : '0');
        
        return exists;
    }
}
```

---

## �🚀 **PERFORMANCE OPTIMIZATIONS:**

### **Redis Caching Strategy:**
```typescript
// src/utils/Cache.ts
import Redis from 'ioredis';

export class Cache {
  private redis: Redis;
  
  constructor() {
    this.redis = new Redis({
      host: process.env.REDIS_HOST,
      port: parseInt(process.env.REDIS_PORT || '6379'),
      password: process.env.REDIS_PASSWORD,
      retryDelayOnFailover: 100,
      enableReadyCheck: true,
      lazyConnect: true
    });
  }
  
  async get(key: string): Promise<any> {
    try {
      const value = await this.redis.get(key);
      return value ? JSON.parse(value) : null;
    } catch (error) {
      console.error('Cache get error:', error);
      return null;
    }
  }
  
  async set(key: string, value: any, ttl: number = 3600): Promise<void> {
    try {
      await this.redis.setex(key, ttl, JSON.stringify(value));
    } catch (error) {
      console.error('Cache set error:', error);
    }
  }
  
  async delete(key: string): Promise<void> {
    try {
      await this.redis.del(key);
    } catch (error) {
      console.error('Cache delete error:', error);
    }
  }
}
```

---

## 📊 **MONITORING & ANALYTICS:**

### **Performance Metrics Collection:**
```typescript
// src/monitoring/URLMetrics.ts
export class URLMetrics {
    private metrics = new Map<string, any>();
    
    /**
     * Track generation performance
     */
    trackGeneration(duration: number, strategy: string, attempts: number, success: boolean) {
        const metric = {
            duration,
            strategy,
            attempts,
            success,
            timestamp: Date.now()
        };
        
        // Store in time-series database (InfluxDB/TimescaleDB)
        this.recordMetric('url_generation', metric);
    }
    
    /**
     * Track lookup performance
     */
    trackLookup(shortCode: string, cacheHit: boolean, duration: number) {
        this.recordMetric('url_lookup', {
            shortCode,
            cacheHit,
            duration,
            timestamp: Date.now()
        });
    }
    
    /**
     * Collision rate monitoring
     */
    trackCollision(codeLength: number, attempt: number, resolved: boolean) {
        this.recordMetric('collision_rate', {
            codeLength,
            attempt,
            resolved,
            timestamp: Date.now()
        });
    }
    
    /**
     * Generate performance report
     */
    async generateReport(timeframe: string = '24h'): Promise<PerformanceReport> {
        const [generation, lookup, collision] = await Promise.all([
            this.getGenerationStats(timeframe),
            this.getLookupStats(timeframe),
            this.getCollisionStats(timeframe)
        ]);
        
        return {
            generation: {
                avgDuration: generation.avgDuration,
                successRate: generation.successRate,
                totalGenerated: generation.count,
                strategyBreakdown: generation.strategies
            },
            lookup: {
                avgDuration: lookup.avgDuration,
                cacheHitRate: lookup.cacheHitRate,
                totalLookups: lookup.count,
                p95Duration: lookup.p95
            },
            collision: {
                collisionRate: collision.rate,
                avgAttempts: collision.avgAttempts,
                resolutionSuccessRate: collision.resolutionRate
            },
            recommendations: this.generateRecommendations(generation, lookup, collision)
        };
    }
    
    private generateRecommendations(gen: any, lookup: any, collision: any): string[] {
        const recommendations = [];
        
        if (collision.rate > 0.01) {
            recommendations.push('Consider increasing short code length to reduce collisions');
        }
        
        if (lookup.cacheHitRate < 0.80) {
            recommendations.push('Optimize cache TTL or increase cache capacity');
        }
        
        if (gen.avgDuration > 100) {
            recommendations.push('Generation performance slow - check database connection pool');
        }
        
        return recommendations;
    }
}
```

### **Real-time Alerts:**
```typescript
// src/monitoring/AlertManager.ts
export class AlertManager {
    
    /**
     * Monitor collision rate in real-time
     */
    monitorCollisionRate() {
        setInterval(async () => {
            const rate = await this.getCollisionRateLastHour();
            
            if (rate > 0.05) { // 5% collision rate threshold
                this.sendAlert('HIGH_COLLISION_RATE', {
                    rate,
                    threshold: 0.05,
                    recommendation: 'Increase short code length immediately'
                });
            }
        }, 300000); // Check every 5 minutes
    }
    
    /**
     * Monitor generation performance
     */
    monitorPerformance() {
        setInterval(async () => {
            const stats = await this.getPerformanceStats();
            
            if (stats.avgGenerationTime > 500) { // 500ms threshold
                this.sendAlert('SLOW_GENERATION', {
                    avgTime: stats.avgGenerationTime,
                    threshold: 500
                });
            }
            
            if (stats.cacheHitRate < 0.70) { // 70% cache hit rate
                this.sendAlert('LOW_CACHE_HIT_RATE', {
                    hitRate: stats.cacheHitRate,
                    threshold: 0.70
                });
            }
        }, 60000); // Check every minute
    }
}
```

---

## 🧪 **COMPREHENSIVE TESTING STRATEGY:**

### **Unit Tests:**
```typescript
// tests/unit/Base62Encoder.test.ts
describe('Base62Encoder', () => {
    
    describe('encode/decode consistency', () => {
        test('should encode and decode numbers correctly', () => {
            const testNumbers = [0, 1, 61, 62, 3844, 238327, 14776336];
            
            testNumbers.forEach(num => {
                const encoded = Base62Encoder.encode(num);
                const decoded = Base62Encoder.decode(encoded);
                expect(decoded).toBe(num);
            });
        });
        
        test('should handle minimum length padding', () => {
            const encoded = Base62Encoder.encode(1, 8);
            expect(encoded).toHaveLength(8);
            expect(encoded).toMatch(/^0+1$/);
        });
    });
    
    describe('validation', () => {
        test('should validate correct Base62 strings', () => {
            expect(Base62Encoder.isValid('abc123')).toBe(true);
            expect(Base62Encoder.isValid('XYZ789')).toBe(true);
        });
        
        test('should reject invalid characters', () => {
            expect(Base62Encoder.isValid('abc-123')).toBe(false);
            expect(Base62Encoder.isValid('abc@123')).toBe(false);
        });
        
        test('should reject invalid lengths', () => {
            expect(Base62Encoder.isValid('a')).toBe(false); // Too short
            expect(Base62Encoder.isValid('a'.repeat(100))).toBe(false); // Too long
        });
    });
});

// tests/unit/ShortCodeGenerator.test.ts
describe('ShortCodeGenerator', () => {
    
    describe('collision handling', () => {
        test('should generate different codes on collision', async () => {
            const codes = new Set();
            
            // Generate 1000 codes
            for (let i = 0; i < 1000; i++) {
                const code = await ShortCodeGenerator.generateRandom();
                expect(codes.has(code)).toBe(false);
                codes.add(code);
            }
        });
        
        test('should escalate length on repeated collisions', async () => {
            // Mock collision scenario
            jest.spyOn(ShortCodeGenerator, 'checkCodeExists')
                .mockResolvedValueOnce(true) // First attempt collision
                .mockResolvedValueOnce(true) // Second attempt collision
                .mockResolvedValueOnce(false); // Third attempt success
            
            const code = await ShortCodeGenerator.generateRandom();
            expect(code).toHaveLength(7); // Should escalate to 7 chars
        });
    });
});
```

### **Integration Tests:**
```typescript
// tests/integration/URLShortening.test.ts
describe('URL Shortening Integration', () => {
    let app: Application;
    let db: Database;
    let redis: Redis;
    
    beforeAll(async () => {
        // Setup test environment
        app = await createTestApp();
        db = await createTestDatabase();
        redis = await createTestRedis();
    });
    
    afterAll(async () => {
        await cleanupTestEnvironment(app, db, redis);
    });
    
    describe('POST /api/shorten', () => {
        test('should create short URL successfully', async () => {
            const response = await request(app)
                .post('/api/shorten')
                .send({
                    url: 'https://example.com/very-long-url-that-needs-shortening'
                })
                .expect(201);
            
            expect(response.body.success).toBe(true);
            expect(response.body.data.shortCode).toMatch(/^[a-zA-Z0-9]{6,}$/);
            expect(response.body.data.shortUrl).toContain(response.body.data.shortCode);
        });
        
        test('should handle custom alias', async () => {
            const customAlias = 'mylink123';
            
            const response = await request(app)
                .post('/api/shorten')
                .send({
                    url: 'https://example.com',
                    customAlias
                })
                .expect(201);
            
            expect(response.body.data.shortCode).toBe(customAlias);
        });
        
        test('should prevent duplicate custom alias', async () => {
            const customAlias = 'duplicate123';
            
            // First request should succeed
            await request(app)
                .post('/api/shorten')
                .send({ url: 'https://example1.com', customAlias })
                .expect(201);
            
            // Second request should fail
            await request(app)
                .post('/api/shorten')
                .send({ url: 'https://example2.com', customAlias })
                .expect(400);
        });
    });
    
    describe('GET /{shortCode}', () => {
        test('should redirect to original URL', async () => {
            const originalUrl = 'https://example.com/target';
            
            // Create short URL
            const createResponse = await request(app)
                .post('/api/shorten')
                .send({ url: originalUrl });
            
            const shortCode = createResponse.body.data.shortCode;
            
            // Test redirect
            const response = await request(app)
                .get(`/${shortCode}`)
                .expect(301);
            
            expect(response.headers.location).toBe(originalUrl);
        });
        
        test('should return 404 for non-existent code', async () => {
            await request(app)
                .get('/nonexistent123')
                .expect(404);
        });
    });
});
```

### **Performance Tests:**
```typescript
// tests/performance/LoadTest.test.ts
describe('Performance Load Tests', () => {
    
    test('should handle concurrent URL creation', async () => {
        const concurrency = 100;
        const urlsPerThread = 10;
        
        const promises = Array.from({ length: concurrency }, async (_, i) => {
            const results = [];
            
            for (let j = 0; j < urlsPerThread; j++) {
                const startTime = Date.now();
                
                const response = await request(app)
                    .post('/api/shorten')
                    .send({
                        url: `https://example.com/test-${i}-${j}`
                    });
                
                const duration = Date.now() - startTime;
                
                results.push({
                    success: response.status === 201,
                    duration,
                    shortCode: response.body.data?.shortCode
                });
            }
            
            return results;
        });
        
        const allResults = (await Promise.all(promises)).flat();
        
        // Verify all requests succeeded
        const successCount = allResults.filter(r => r.success).length;
        expect(successCount).toBe(concurrency * urlsPerThread);
        
        // Verify performance
        const avgDuration = allResults.reduce((sum, r) => sum + r.duration, 0) / allResults.length;
        expect(avgDuration).toBeLessThan(500); // 500ms average
        
        // Verify uniqueness
        const codes = allResults.map(r => r.shortCode);
        const uniqueCodes = new Set(codes);
        expect(uniqueCodes.size).toBe(codes.length);
    });
    
    test('should handle high redirect load', async () => {
        // Pre-create URLs for testing
        const testUrls = [];
        for (let i = 0; i < 50; i++) {
            const response = await request(app)
                .post('/api/shorten')
                .send({ url: `https://example.com/target-${i}` });
            testUrls.push(response.body.data.shortCode);
        }
        
        // Test concurrent redirects
        const concurrency = 200;
        const redirectsPerThread = 25;
        
        const promises = Array.from({ length: concurrency }, async () => {
            const results = [];
            
            for (let j = 0; j < redirectsPerThread; j++) {
                const shortCode = testUrls[Math.floor(Math.random() * testUrls.length)];
                const startTime = Date.now();
                
                const response = await request(app)
                    .get(`/${shortCode}`)
                    .redirects(0); // Don't follow redirects
                
                const duration = Date.now() - startTime;
                
                results.push({
                    success: response.status === 301,
                    duration
                });
            }
            
            return results;
        });
        
        const allResults = (await Promise.all(promises)).flat();
        
        // Performance assertions
        const successCount = allResults.filter(r => r.success).length;
        expect(successCount).toBe(concurrency * redirectsPerThread);
        
        const avgRedirectTime = allResults.reduce((sum, r) => sum + r.duration, 0) / allResults.length;
        expect(avgRedirectTime).toBeLessThan(50); // 50ms average redirect
        
        const p95RedirectTime = allResults
            .map(r => r.duration)
            .sort((a, b) => a - b)[Math.floor(allResults.length * 0.95)];
        expect(p95RedirectTime).toBeLessThan(100); // 100ms P95 redirect
    });
});
```

---

## ✅ **ENHANCED ACCEPTANCE CRITERIA:**

### **Functional Requirements:**
- [ ] **Core Functionality**:
  - [ ] Generate unique 6-character Base62 short codes
  - [ ] Support custom aliases (3-50 characters)
  - [ ] Validate and normalize input URLs
  - [ ] Handle URL expiration correctly
  - [ ] Support password protection

- [ ] **Performance Requirements**:
  - [ ] URL generation < 200ms (P95)
  - [ ] URL redirect < 50ms (P95)
  - [ ] Support 1,000+ concurrent requests
  - [ ] Cache hit rate > 80% for redirects
  - [ ] Database queries < 10ms (average)

- [ ] **Collision Handling**:
  - [ ] Collision rate < 1% for 6-character codes
  - [ ] Automatic length escalation on repeated collisions
  - [ ] Maximum 5 retry attempts before failure
  - [ ] Collision monitoring and alerting

### **Technical Requirements:**
- [ ] **Database Optimization**:
  - [ ] Proper indexing strategy implemented
  - [ ] Batch operations for bulk checks
  - [ ] Connection pooling configured
  - [ ] Query performance monitoring

- [ ] **Caching Strategy**:
  - [ ] Multi-tier Redis caching
  - [ ] Negative caching for 404 prevention
  - [ ] Cache invalidation on updates
  - [ ] Dynamic TTL based on access patterns

- [ ] **Security & Validation**:
  - [ ] Input sanitization and validation
  - [ ] Rate limiting implementation
  - [ ] SQL injection prevention
  - [ ] XSS protection for URLs

### **Quality Assurance:**
- [ ] **Test Coverage**:
  - [ ] Unit tests > 90% coverage
  - [ ] Integration tests for all API endpoints
  - [ ] Performance tests under load
  - [ ] Security penetration testing

- [ ] **Monitoring & Alerting**:
  - [ ] Real-time metrics collection
  - [ ] Performance dashboard
  - [ ] Alert thresholds configured
  - [ ] Error tracking integration

- [ ] **Documentation**:
  - [ ] API documentation complete
  - [ ] Database schema documented
  - [ ] Algorithm explanation detailed
  - [ ] Troubleshooting guide available

---

## 📊 **SUCCESS METRICS:**

### **Performance Benchmarks:**
```yaml
URL Generation:
  - Average Response Time: < 100ms
  - P95 Response Time: < 200ms
  - P99 Response Time: < 500ms
  - Success Rate: > 99.9%
  
URL Redirection:
  - Average Response Time: < 25ms
  - P95 Response Time: < 50ms
  - P99 Response Time: < 100ms
  - Cache Hit Rate: > 80%
  
Collision Management:
  - Collision Rate: < 1% (6-char codes)
  - Resolution Success Rate: > 99.5%
  - Average Retry Attempts: < 1.5
  
Database Performance:
  - Query Response Time: < 10ms average
  - Connection Pool Utilization: < 80%
  - Index Hit Rate: > 99%
```

### **Scalability Targets:**
```yaml
Concurrent Users: 10,000+
Requests per Second: 1,000+
Daily URL Creation: 100,000+
Total URLs Supported: 10,000,000+ (with current 6-char strategy)
Uptime SLA: 99.9%
Data Durability: 99.999%
```

---

## 🚀 **DEPLOYMENT CHECKLIST:**
- [ ] Database migration scripts tested
- [ ] Redis cluster configuration verified  
- [ ] Environment variables configured
- [ ] Load balancer health checks configured
- [ ] Monitoring dashboards deployed
- [ ] Alert thresholds configured
- [ ] Backup and recovery procedures tested
- [ ] Performance baselines established

---

## 📝 **COMPLETION CRITERIA:**
- [ ] All acceptance criteria met and verified
- [ ] Performance benchmarks achieved
- [ ] Security review completed and passed
- [ ] Code review approved by senior developer
- [ ] Documentation complete and reviewed
- [ ] Monitoring and alerts operational
- [ ] Production deployment successful
- [ ] Post-deployment verification completed

---

**🎆 Task Status**: ✅ **COMPLETED**  
**Completion Date**: _________  
**Review By**: Senior Backend Developer + Tech Lead  
**Next Task**: [URL Redirect Optimization](task-002-url-redirect-optimization.md)