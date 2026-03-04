# TASK: URL Validation & Security

**Ticket**: Core URL Shortening  
**Priority**: P0 (CRITICAL - Security)  
**Assignee**: Security Engineer + Backend Developer  
**Estimate**: 2 days  
**Dependencies**: URL Shortening Engine  

## 📋 TASK OVERVIEW

**Objective**: Implement comprehensive URL validation និង security measures  
**Success Criteria**: Block malicious URLs និង ensure platform security  

---

## 🎯 **SECURITY REQUIREMENTS:**

### **URL Validation:**
- [ ] Malicious URL detection (malware, phishing)
- [ ] Content-type verification
- [ ] Domain reputation checking
- [ ] URL sanitization និង normalization
- [ ] Blacklist management system

### **Input Security:**
- [ ] SQL injection prevention
- [ ] XSS protection
- [ ] CSRF protection
- [ ] Rate limiting per IP/user
- [ ] Input length limits

### **URL Content Analysis:**
- [ ] Safe browsing API integration
- [ ] Real-time threat detection
- [ ] Content scanning for violations
- [ ] Adult content filtering
- [ ] Spam URL detection

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **URL Validator Service:**
```typescript
// src/services/URLValidatorService.ts
import axios from 'axios';
import { URL } from 'url';
import { Cache } from '../utils/Cache';

export class URLValidatorService {
  private cache = new Cache();
  private safeBrowsingAPI: SafeBrowsingAPI;
  private virusTotalAPI: VirusTotalAPI;
  
  constructor() {
    this.safeBrowsingAPI = new SafeBrowsingAPI(process.env.GOOGLE_SAFE_BROWSING_KEY);
    this.virusTotalAPI = new VirusTotalAPI(process.env.VIRUSTOTAL_API_KEY);
  }
  
  /**
   * Comprehensive URL validation and security checking
   */
  async validateURL(originalUrl: string): Promise<URLValidationResult> {
    const startTime = performance.now();
    
    try {
      // 1. Basic format validation
      const normalizedUrl = this.normalizeURL(originalUrl);
      
      // 2. Check cached validation results
      const cachedResult = await this.getCachedValidation(normalizedUrl);
      if (cachedResult) {
        return cachedResult;
      }
      
      // 3. Parallel security checks
      const [
        formatCheck,
        blacklistCheck,
        safeBrowsingCheck,
        contentCheck,
        reputationCheck
      ] = await Promise.allSettled([
        this.validateFormat(normalizedUrl),
        this.checkBlacklist(normalizedUrl),
        this.checkSafeBrowsing(normalizedUrl),
        this.checkContent(normalizedUrl),
        this.checkDomainReputation(normalizedUrl)
      ]);
      
      // 4. Aggregate results
      const result = this.aggregateValidationResults({
        formatCheck,
        blacklistCheck,
        safeBrowsingCheck,
        contentCheck,
        reputationCheck
      });
      
      // 5. Cache results for performance
      await this.cacheValidationResult(normalizedUrl, result);
      
      // 6. Log security events
      if (!result.isValid) {
        await this.logSecurityEvent(normalizedUrl, result);
      }
      
      const validationTime = performance.now() - startTime;
      console.log(`URL validation completed in ${validationTime.toFixed(2)}ms`);
      
      return result;
      
    } catch (error) {
      console.error('URL validation error:', error);
      
      // Fail securely: reject if validation fails
      return {
        isValid: false,
        normalizedUrl: originalUrl,
        threats: ['VALIDATION_ERROR'],
        riskScore: 100,
        reason: 'Validation service error'
      };
    }
  }
  
  /**
   * Normalize and sanitize URL
   */
  private normalizeURL(url: string): string {
    try {
      // Remove dangerous protocols         
      if (!/^https?:\/\//i.test(url)) {
        if (url.startsWith('//')) {
          url = 'https:' + url;
        } else if (!url.startsWith('http')) {
          url = 'https://' + url;
        }
      }
      
      const urlObj = new URL(url);
      
      // Security checks on protocol
      if (!['http:', 'https:'].includes(urlObj.protocol)) {
        throw new Error('Invalid protocol');
      }
      
      // Normalize
      urlObj.pathname = decodeURIComponent(urlObj.pathname);
      urlObj.hostname = urlObj.hostname.toLowerCase();
      
      // Remove tracking parameters (privacy)
      const trackingParams = ['utm_source', 'utm_medium', 'utm_campaign', 'fbclid', 'gclid'];
      trackingParams.forEach(param => {
        urlObj.searchParams.delete(param);
      });
      
      return urlObj.toString();
      
    } catch (error) {
      throw new Error('Invalid URL format');
    }
  }
  
  /**
   * Basic format and structure validation
   */
  private async validateFormat(url: string): Promise<ValidationCheck> {
    try {
      const urlObj = new URL(url);
      
      // Check for suspicious patterns
      const suspiciousPatterns = [
        /[а-я]/i, // Cyrillic characters (IDN homograph attacks)
        /[\u0080-\uFFFF]/, // Non-ASCII characters
        /bit\.ly|tinyurl\.com|t\.co/i, // Existing shorteners (loop prevention)
        /%[0-9a-f]{2}/i, // URL encoding (potential obfuscation)
      ];
      
      const threats = [];
      
      for (const pattern of suspiciousPatterns) {
        if (pattern.test(url)) {
          threats.push('SUSPICIOUS_PATTERN');
          break;
        }
      }
      
      // Check hostname validity 
      if (urlObj.hostname.length > 253) {
        threats.push('INVALID_HOSTNAME');
      }
      
      if (urlObj.pathname.length > 2000) {
        threats.push('EXCESSIVE_PATH_LENGTH');
      }
      
      return {
        isValid: threats.length === 0,
        threats,
        riskScore: threats.length * 20
      };
      
    } catch (error) {
      return {
        isValid: false,
        threats: ['INVALID_FORMAT'],
        riskScore: 100
      };
    }
  }
  
  /**
   * Check against known blacklists
   */
  private async checkBlacklist(url: string): Promise<ValidationCheck> {
    try {
      const urlObj = new URL(url);
      const domain = urlObj.hostname;
      
      // Check internal blacklist
      const isBlacklisted = await this.isInBlacklist(domain);
      
      if (isBlacklisted) {
        return {
          isValid: false,
          threats: ['BLACKLISTED_DOMAIN'],
          riskScore: 100
        };
      }
      
      // Check popular blacklist APIs
      const blacklistChecks = await Promise.allSettled([
        this.checkSpamhausBlacklist(domain),
        this.checkPhishtankBlacklist(url),
        this.checkMalwareDomainList(domain)
      ]);
      
      const threats = [];
      let maxRiskScore = 0;
      
      blacklistChecks.forEach(result => {
        if (result.status === 'fulfilled' && !result.value.isValid) {
          threats.push(...result.value.threats);
          maxRiskScore = Math.max(maxRiskScore, result.value.riskScore);
        }
      });
      
      return {
        isValid: threats.length === 0,
        threats,
        riskScore: maxRiskScore
      };
      
    } catch (error) {
      console.error('Blacklist check error:', error);
      return {
        isValid: true, // Don't block on API failure
        threats: [],
        riskScore: 0
      };
    }
  }
  
  /**
   * Google Safe Browsing API check
   */
  private async checkSafeBrowsing(url: string): Promise<ValidationCheck> {
    try {
      const result = await this.safeBrowsingAPI.checkURL(url);
      
      if (result.threats.length > 0) {
        return {
          isValid: false,
          threats: result.threats,
          riskScore: 95
        };
      }
      
      return {
        isValid: true,
        threats: [],
        riskScore: 0
      };
      
    } catch (error) {
      console.error('Safe Browsing check error:', error);
      return {
        isValid: true, // Don't block on API failure
        threats: [],
        riskScore: 0
      };
    }
  }
  
  /**
   * Content analysis (scan actual page content)
   */
  private async checkContent(url: string): Promise<ValidationCheck> {
    try {
      // Timeout for content check
      const timeout = 5000; // 5 seconds
      
      const response = await axios.get(url, {
        timeout,
        maxRedirects: 3,
        validateStatus: (status) => status < 400,
        headers: {
          'User-Agent': 'ShortLink-Bot/1.0 (+https://shortlink.com/bot)'
        }
      });
      
      const contentType = response.headers['content-type'] || '';
      const content = response.data;
      
      const threats = [];
      let riskScore = 0;
      
      // Check content type
      if (!contentType.includes('text/html')) {
        if (contentType.includes('application/') && !contentType.includes('json')) {
          threats.push('EXECUTABLE_CONTENT');
          riskScore += 30;
        }
      }
      
      // Scan HTML content for malicious patterns
      if (typeof content === 'string') {
        const maliciousPatterns = [
          /<script[^>]*>.*?(phishing|malware|virus)/i,
          /document\.write\s*\(/i,
          /eval\s*\(/i,
          /window\.location\s*=/i,
          /<iframe[^>]*src=[^>]*>/i
        ];
        
        maliciousPatterns.forEach(pattern => {
          if (pattern.test(content)) {
            threats.push('MALICIOUS_CONTENT');
            riskScore += 20;
          }
        });
        
        // Check for phishing keywords
        const phishingKeywords = [
          'verify your account', 'suspended account', 'click here immediately',
          'urgent action required', 'confirm your identity', 'security alert'
        ];
        
        const lowercaseContent = content.toLowerCase();
        phishingKeywords.forEach(keyword => {
          if (lowercaseContent.includes(keyword)) {
            threats.push('PHISHING_CONTENT');
            riskScore += 15;
          }
        });
      }
      
      return {
        isValid: riskScore < 50,
        threats,
        riskScore: Math.min(riskScore, 100)
      };
      
    } catch (error) {
      // Don't block URLs if content check fails
      return {
        isValid: true,
        threats: [],
        riskScore: 0
      };
    }
  }
  
  /**
   * Domain reputation checking
   */
  private async checkDomainReputation(url: string): Promise<ValidationCheck> {
    try {
      const urlObj = new URL(url);
      const domain = urlObj.hostname;
      
      // Check domain age and reputation
      const [ageCheck, reputationCheck] = await Promise.allSettled([
        this.checkDomainAge(domain),
        this.checkDomainReputation(domain)
      ]);
      
      let riskScore = 0;
      const threats = [];
      
      // Very new domains are riskier
      if (ageCheck.status === 'fulfilled' && ageCheck.value.ageInDays < 7) {
        threats.push('NEW_DOMAIN');
        riskScore += 30;
      }
      
      // Poor reputation
      if (reputationCheck.status === 'fulfilled' && reputationCheck.value.score < 50) {
        threats.push('POOR_REPUTATION');
        riskScore += reputationCheck.value.riskScore;
      }
      
      return {
        isValid: riskScore < 70,
        threats,
        riskScore
      };
      
    } catch (error) {
      return {
        isValid: true,
        threats: [],
        riskScore: 0
      };
    }
  }
  
  /**
   * Cache validation results for performance
   */
  private async cacheValidationResult(url: string, result: URLValidationResult): Promise<void> {
    const cacheKey = `url_validation:${Buffer.from(url).toString('base64')}`;
    const ttl = result.isValid ? 3600 : 1800; // Cache valid URLs longer
    
    await this.cache.set(cacheKey, result, ttl);
  }
  
  /**
   * Get cached validation result
   */
  private async getCachedValidation(url: string): Promise<URLValidationResult | null> {
    const cacheKey = `url_validation:${Buffer.from(url).toString('base64')}`;
    return await this.cache.get(cacheKey);
  }
  
  /**
   * Log security events for monitoring
   */
  private async logSecurityEvent(url: string, result: URLValidationResult): Promise<void> {
    const event = {
      type: 'URL_VALIDATION_FAILED',
      url: url,
      threats: result.threats,
      riskScore: result.riskScore,
      timestamp: new Date(),
      severity: result.riskScore > 80 ? 'HIGH' : 'MEDIUM'
    };
    
    // Log to security monitoring system
    console.warn('Security Event:', event);
    
    // Store in database for analysis
    // await SecurityEvent.create(event);
  }
}

interface ValidationCheck {
  isValid: boolean;
  threats: string[];
  riskScore: number;
}

interface URLValidationResult {
  isValid: boolean;
  normalizedUrl: string;
  threats: string[];
  riskScore: number;
  reason?: string;
}
```

### **Safe Browsing API Integration:**
```typescript
// src/services/SafeBrowsingAPI.ts
import axios from 'axios';

export class SafeBrowsingAPI {
  constructor(private apiKey: string) {}
  
  async checkURL(url: string): Promise<{ threats: string[] }> {
    try {
      const response = await axios.post(
        `https://safebrowsing.googleapis.com/v4/threatMatches:find?key=${this.apiKey}`,
        {
          client: {
            clientId: 'shortlink-platform',
            clientVersion: '1.0.0'
          },
          threatInfo: {
            threatTypes: [
              'MALWARE',
              'SOCIAL_ENGINEERING',
              'UNWANTED_SOFTWARE',
              'POTENTIALLY_HARMFUL_APPLICATION'
            ],
            platformTypes: ['ANY_PLATFORM'],
            threatEntryTypes: ['URL'],
            threatEntries: [{ url }]
          }
        },
        {
          timeout: 3000,
          headers: {
            'Content-Type': 'application/json'
          }
        }
      );
      
      if (response.data.matches && response.data.matches.length > 0) {
        const threats = response.data.matches.map((match: any) => match.threatType);
        return { threats };
      }
      
      return { threats: [] };
      
    } catch (error) {
      console.error('Safe Browsing API error:', error);
      return { threats: [] }; // Fail open for availability
    }
  }
}
```

### **Security Middleware:**
```typescript
// src/middleware/SecurityMiddleware.ts
import { Request, Response, NextFunction } from 'express';
import rateLimit from 'express-rate-limit';
import helmet from 'helmet';
import { URLValidatorService } from '../services/URLValidatorService';

export class SecurityMiddleware {
  private urlValidator = new URLValidatorService();
  
  /**
   * Rate limiting configuration
   */
  static createRateLimit() {
    return rateLimit({
      windowMs: 15 * 60 * 1000, // 15 minutes
      max: 100, // limit each IP to 100 requests per windowMs
      message: {
        success: false,
        error: 'Too many requests, please try again later.',
        retryAfter: 15 * 60
      },
      standardHeaders: true,
      legacyHeaders: false,
      // Skip successful responses from rate limiting
      skip: (req: Request, res: Response) => res.statusCode < 400
    });
  }
  
  /**
   * Security headers middleware
   */
  static setupSecurityHeaders() {
    return helmet({
      contentSecurityPolicy: {
        directives: {
          defaultSrc: ["'self'"],
          styleSrc: ["'self'", "'unsafe-inline'"],
          scriptSrc: ["'self'"],
          imgSrc: ["'self'", "data:", "https:"],
          connectSrc: ["'self'"],
          fontSrc: ["'self'"],
          objectSrc: ["'none'"],
          mediaSrc: ["'self'"],
          frameSrc: ["'none'"],
        },
      },
      crossOriginEmbedderPolicy: false
    });
  }
  
  /**
   * URL validation middleware
   */
  validateURL() {
    return async (req: Request, res: Response, next: NextFunction) => {
      try {
        const { url } = req.body;
        
        if (!url) {
          return next();
        }
        
        const validationResult = await this.urlValidator.validateURL(url);
        
        if (!validationResult.isValid) {
          return res.status(400).json({
            success: false,
            error: 'URL validation failed',
            details: {
              threats: validationResult.threats,
              riskScore: validationResult.riskScore,
              reason: validationResult.reason
            }
          });
        }
        
        // Add normalized URL to request
        req.body.originalUrl = validationResult.normalizedUrl;
        next();
        
      } catch (error) {
        console.error('URL validation middleware error:', error);
        
        res.status(400).json({
          success: false,
          error: 'URL validation failed',
          message: 'Please check your URL and try again'
        });
      }
    };
  }
  
  /**
   * Input sanitization middleware
   */
  static sanitizeInput() {
    return (req: Request, res: Response, next: NextFunction) => {
      // Recursively sanitize all string inputs
      const sanitizeObject = (obj: any): any => {
        if (typeof obj === 'string') {
          // Remove potential XSS vectors
          return obj
            .replace(/[<>]/g, '') // Remove angle brackets
            .replace(/javascript:/gi, '') // Remove javascript: protocol
            .replace(/on\w+=/gi, '') // Remove event handlers
            .trim();
        }
        
        if (Array.isArray(obj)) {
          return obj.map(sanitizeObject);
        }
        
        if (obj && typeof obj === 'object') {
          const sanitized: any = {};
          for (const [key, value] of Object.entries(obj)) {
            sanitized[key] = sanitizeObject(value);
          }
          return sanitized;
        }
        
        return obj;
      };
      
      req.body = sanitizeObject(req.body);
      req.query = sanitizeObject(req.query);
      req.params = sanitizeObject(req.params);
      
      next();
    };
  }
}
```

---

## 🔐 **SECURITY CONFIGURATIONS:**

### **Environment Security:**
```bash
# Security environment variables
GOOGLE_SAFE_BROWSING_KEY=your_safe_browsing_api_key
VIRUSTOTAL_API_KEY=your_virustotal_api_key
JWT_SECRET=your_very_long_and_random_jwt_secret
RATE_LIMIT_REDIS_URL=redis://localhost:6379/1

# Security settings
MAX_URL_LENGTH=2000
MAX_CUSTOM_ALIAS_LENGTH=50
VALIDATION_CACHE_TTL=3600
SECURITY_LOG_LEVEL=warn
```

### **Blacklist Management:**
```sql
-- Security blacklist tables
CREATE TABLE domain_blacklist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain VARCHAR(255) NOT NULL UNIQUE,
    reason VARCHAR(500),
    severity VARCHAR(20) DEFAULT 'HIGH',
    added_by UUID REFERENCES users(id),
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_domain_blacklist_domain (domain)
);

-- URL pattern blacklist
CREATE TABLE url_pattern_blacklist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pattern TEXT NOT NULL,
    regex_pattern BOOLEAN DEFAULT FALSE,
    reason VARCHAR(500),
    severity VARCHAR(20) DEFAULT 'MEDIUM',
    added_by UUID REFERENCES users(id),
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Security events log
CREATE TABLE security_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    url TEXT,
    ip_address INET,
    user_id UUID REFERENCES users(id),
    threat_types TEXT[],
    risk_score INTEGER,
    severity VARCHAR(20),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_security_events_type (event_type),
    INDEX idx_security_events_created (created_at),
    INDEX idx_security_events_severity (severity)
);
```

---

## ✅ **ACCEPTANCE CRITERIA:**
- [ ] URL validation blocking malicious URLs (>95% accuracy)
- [ ] Rate limiting preventing abuse
- [ ] Input sanitization preventing XSS/injection
- [ ] Security headers implemented correctly
- [ ] Blacklist management functional
- [ ] Real-time threat detection working
- [ ] Security event logging active
- [ ] Performance impact < 50ms per validation
- [ ] False positive rate < 2%
- [ ] Security monitoring dashboard functional

---

## 🧪 **TESTING CHECKLIST:**
- [ ] Malicious URL detection tests
- [ ] Rate limiting stress tests  
- [ ] Input sanitization security tests
- [ ] XSS prevention validation
- [ ] SQL injection prevention tests
- [ ] Security header verification
- [ ] Blacklist functionality tests
- [ ] Performance impact assessment

---

**Completion Date**: _________  
**Review By**: Security Engineer + Senior Backend  
**Next Task**: Basic Analytics Tracking