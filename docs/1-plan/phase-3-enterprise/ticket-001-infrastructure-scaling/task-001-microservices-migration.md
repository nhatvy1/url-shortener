# TASK: Microservices Architecture Migration

**Ticket**: Infrastructure Scaling  
**Priority**: P0 (Critical Architecture)  
**Assignee**: DevOps Lead + Senior Backend Developer  
**Estimate**: 8 days  
**Dependencies**: Current Monolith Application  

## 📋 TASK OVERVIEW

**Objective**: Migrate monolithic architecture to scalable microservices  
**Success Criteria**: Independent services deployed with 99.99% uptime  

---

## 🎯 **MICROSERVICES REQUIREMENTS:**

### **Service Architecture:**
- [ ] **URL Service**: Core shortening logic និង redirect handling
- [ ] **User Service**: Authentication, authorization, user management
- [ ] **Analytics Service**: Click tracking, reporting, data processing  
- [ ] **Subscription Service**: Billing, plans, usage tracking
- [ ] **Notification Service**: Email, SMS, push notifications
- [ ] **API Gateway**: Request routing, rate limiting, authentication

### **Communication Patterns:**
- [ ] **Synchronous**: REST APIs with OpenAPI specifications
- [ ] **Asynchronous**: Event-driven architecture with Apache Kafka
- [ ] **Data Consistency**: Saga pattern for distributed transactions
- [ ] **Service Discovery**: Kubernetes native service discovery
- [ ] **Load Balancing**: Application-level load balancing

### **Data Management:**
- [ ] **Database Per Service**: Independent data stores
- [ ] **Shared Data Access**: Event-driven synchronization
- [ ] **CQRS Implementation**: Command Query Responsibility Segregation
- [ ] **Event Sourcing**: Audit trail និង data replay capability

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **Service Architecture Design:**
```yaml
# docker-compose.microservices.yml
version: '3.8'

services:
  # API Gateway (Kong)
  api-gateway:
    image: kong:latest
    container_name: shortlink-gateway
    ports:
      - "8000:8000"  # Proxy
      - "8001:8001"  # Admin API
    environment:
      KONG_DATABASE: "off"
      KONG_DECLARATIVE_CONFIG: /kong/declarative/kong.yml
    volumes:
      - ./kong/kong.yml:/kong/declarative/kong.yml
    networks:
      - shortlink-network

  # URL Shortening Service
  url-service:
    build:
      context: ./services/url-service
      dockerfile: Dockerfile
    container_name: shortlink-url-service
    ports:
      - "3001:3000"
    environment:
      - DATABASE_URL=${URL_SERVICE_DB_URL}
      - REDIS_URL=${REDIS_URL}
      - KAFKA_BROKERS=${KAFKA_BROKERS}
    depends_on:
      - postgres-url
      - redis
      - kafka
    networks:
      - shortlink-network

  # User Management Service  
  user-service:
    build:
      context: ./services/user-service
      dockerfile: Dockerfile
    container_name: shortlink-user-service
    ports:
      - "3002:3000"
    environment:
      - DATABASE_URL=${USER_SERVICE_DB_URL}
      - JWT_SECRET=${JWT_SECRET}
      - KAFKA_BROKERS=${KAFKA_BROKERS}
    depends_on:
      - postgres-user
      - kafka
    networks:
      - shortlink-network

  # Analytics Service
  analytics-service:
    build:
      context: ./services/analytics-service
      dockerfile: Dockerfile
    container_name: shortlink-analytics-service
    ports:
      - "3003:3000"
    environment:
      - DATABASE_URL=${ANALYTICS_SERVICE_DB_URL}
      - CLICKHOUSE_URL=${CLICKHOUSE_URL}
      - KAFKA_BROKERS=${KAFKA_BROKERS}
    depends_on:
      - clickhouse
      - kafka
    networks:
      - shortlink-network

  # Subscription Service
  subscription-service:
    build:
      context: ./services/subscription-service
      dockerfile: Dockerfile
    container_name: shortlink-subscription-service
    ports:
      - "3004:3000"
    environment:
      - DATABASE_URL=${SUBSCRIPTION_SERVICE_DB_URL}
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - KAFKA_BROKERS=${KAFKA_BROKERS}
    depends_on:
      - postgres-subscription
      - kafka
    networks:
      - shortlink-network

networks:
  shortlink-network:
    driver: bridge
```

### **URL Service Implementation:**
```typescript
// services/url-service/src/app.ts
import express from 'express';
import { URLController } from './controllers/URLController';
import { KafkaProducer } from './messaging/KafkaProducer';
import { HealthCheck } from './middleware/HealthCheck';

export class URLServiceApp {
  private app: express.Application;
  private kafkaProducer: KafkaProducer;
  
  constructor() {
    this.app = express();
    this.kafkaProducer = new KafkaProducer({
      brokers: process.env.KAFKA_BROKERS?.split(',') || ['localhost:9092'],
      clientId: 'url-service'
    });
    
    this.setupMiddleware();
    this.setupRoutes();
    this.setupErrorHandling();
  }
  
  private setupMiddleware(): void {
    this.app.use(express.json({ limit: '10mb' }));
    this.app.use(express.urlencoded({ extended: true }));
    
    // Service-specific middleware
    this.app.use('/health', new HealthCheck().middleware());
    this.app.use('/metrics', this.metricsMiddleware());
  }
  
  private setupRoutes(): void {
    const urlController = new URLController(this.kafkaProducer);
    
    // URL Management Routes
    this.app.post('/urls', urlController.createURL.bind(urlController));
    this.app.get('/urls/:id', urlController.getURL.bind(urlController));
    this.app.put('/urls/:id', urlController.updateURL.bind(urlController));
    this.app.delete('/urls/:id', urlController.deleteURL.bind(urlController));
    
    // Redirect Route (High Performance)
    this.app.get('/:shortCode', urlController.redirect.bind(urlController));
    
    // Bulk Operations
    this.app.post('/urls/bulk', urlController.bulkCreate.bind(urlController));
    this.app.get('/users/:userId/urls', urlController.getUserURLs.bind(urlController));
  }
  
  private metricsMiddleware() {
    return (req: express.Request, res: express.Response) => {
      // Prometheus metrics endpoint
      const metrics = this.collectServiceMetrics();
      res.set('Content-Type', 'text/plain');
      res.send(metrics);
    };
  }
  
  private collectServiceMetrics(): string {
    // Collect and format metrics for Prometheus
    return `
# HELP url_service_requests_total Total number of requests
# TYPE url_service_requests_total counter  
url_service_requests_total{method="POST",endpoint="/urls"} 1234
url_service_requests_total{method="GET",endpoint="/redirect"} 98765

# HELP url_service_response_time Response time in milliseconds
# TYPE url_service_response_time histogram
url_service_response_time_bucket{le="10"} 5432
url_service_response_time_bucket{le="50"} 8901
url_service_response_time_bucket{le="100"} 9876
`;
  }
  
  async start(port: number = 3000): Promise<void> {
    await this.kafkaProducer.connect();
    
    this.app.listen(port, () => {
      console.log(`URL Service listening on port ${port}`);
      console.log(`Health check: http://localhost:${port}/health`);
      console.log(`Metrics: http://localhost:${port}/metrics`);
    });
  }
}

// Start service
const app = new URLServiceApp();
app.start(parseInt(process.env.PORT || '3000'));
```

### **Event-Driven Architecture:**
```typescript
// shared/events/EventTypes.ts
export interface URLCreatedEvent {
  type: 'URL_CREATED';
  data: {
    urlId: string;
    userId: string;
    shortCode: string;
    originalUrl: string;
    createdAt: string;
  };
  metadata: {
    timestamp: string;
    version: string;
    correlationId: string;
  };
}

export interface URLClickedEvent {
  type: 'URL_CLICKED';
  data: {
    urlId: string;
    shortCode: string;
    ipAddress: string;
    userAgent: string;
    referer?: string;
    country?: string;
    deviceType: string;
    clickedAt: string;
  };
  metadata: {
    timestamp: string;
    version: string;
    correlationId: string;
  };
}

export interface UserSubscriptionChangedEvent {
  type: 'USER_SUBSCRIPTION_CHANGED';
  data: {
    userId: string;
    oldPlan: string;
    newPlan: string;
    effectiveDate: string;
    quotaLimits: {
      urlsPerMonth: number;
      apiCallsPerDay: number;
    };
  };
  metadata: {
    timestamp: string;
    version: string;
    correlationId: string;
  };
}
```

### **Service Communication Layer:**
```typescript
// shared/communication/ServiceClient.ts
import axios, { AxiosInstance } from 'axios';
import { CircuitBreaker } from './CircuitBreaker';

export class ServiceClient {
  private client: AxiosInstance;
  private circuitBreaker: CircuitBreaker;
  
  constructor(private serviceName: string, private baseURL: string) {
    this.client = axios.create({
      baseURL,
      timeout: 5000,
      headers: {
        'Content-Type': 'application/json',
        'X-Service-Name': serviceName
      }
    });
    
    this.circuitBreaker = new CircuitBreaker(serviceName, {
      failureThreshold: 5,
      timeout: 3000,
      resetTimeout: 30000
    });
    
    this.setupInterceptors();
  }
  
  private setupInterceptors(): void {
    // Request interceptor for correlation ID
    this.client.interceptors.request.use((config) => {
      config.headers['X-Correlation-ID'] = this.generateCorrelationId();
      config.headers['X-Request-Timestamp'] = new Date().toISOString();
      return config;
    });
    
    // Response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        console.error(`${this.serviceName} service error:`, error.message);
        return Promise.reject(error);
      }
    );
  }
  
  async get<T>(path: string, params?: any): Promise<T> {
    return this.circuitBreaker.execute(async () => {
      const response = await this.client.get(path, { params });
      return response.data;
    });
  }
  
  async post<T>(path: string, data?: any): Promise<T> {
    return this.circuitBreaker.execute(async () => {
      const response = await this.client.post(path, data);
      return response.data;
    });
  }
  
  async put<T>(path: string, data?: any): Promise<T> {
    return this.circuitBreaker.execute(async () => {
      const response = await this.client.put(path, data);
      return response.data;
    });
  }
  
  async delete<T>(path: string): Promise<T> {
    return this.circuitBreaker.execute(async () => {
      const response = await this.client.delete(path);
      return response.data;
    });
  }
  
  private generateCorrelationId(): string {
    return `${this.serviceName}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Service registry for managing service communication
export class ServiceRegistry {
  private static services: Map<string, ServiceClient> = new Map();
  
  static register(serviceName: string, baseURL: string): void {
    this.services.set(serviceName, new ServiceClient(serviceName, baseURL));
  }
  
  static getService(serviceName: string): ServiceClient {
    const service = this.services.get(serviceName);
    if (!service) {
      throw new Error(`Service ${serviceName} not registered`);
    }
    return service;
  }
  
  static async initialize(): Promise<void> {
    // Register all microservices
    this.register('user-service', process.env.USER_SERVICE_URL || 'http://user-service:3000');
    this.register('analytics-service', process.env.ANALYTICS_SERVICE_URL || 'http://analytics-service:3000');
    this.register('subscription-service', process.env.SUBSCRIPTION_SERVICE_URL || 'http://subscription-service:3000');
  }
}
```

### **Kubernetes Deployment Configuration:**
```yaml
# k8s/url-service-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: url-service
  labels:
    app: url-service
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: url-service
  template:
    metadata:
      labels:
        app: url-service
        version: v1
    spec:
      containers:
      - name: url-service
        image: shortlink/url-service:latest
        ports:
        - containerPort: 3000
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: url-service-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: redis-secrets
              key: redis-url
        - name: KAFKA_BROKERS
          value: "kafka-service:9092"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: url-service
spec:
  selector:
    app: url-service
  ports:
  - protocol: TCP
    port: 80
    targetPort: 3000
  type: ClusterIP

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: url-service-ingress
  annotations:
    kubernetes.io/ingress.class: "nginx"
    nginx.ingress.kubernetes.io/rate-limit: "1000"
spec:
  rules:
  - host: api.shortlink.com
    http:
      paths:
      - path: /urls
        pathType: Prefix
        backend:
          service:
            name: url-service
            port:
              number: 80
```

---

## 🔧 **DATA MIGRATION STRATEGY:**

### **Zero-Downtime Migration:**
```typescript
// migration/MonolithToMicroservices.ts
export class MigrationService {
  
  /**
   * Phase 1: Strangler Fig Pattern Implementation
   */
  async implementStranglerFig(): Promise<void> {
    // 1. Create API Gateway routing
    await this.setupGateway();
    
    // 2. Gradually route traffic to new services
    await this.configureTrafficSplitting({
      'url-service': { percentage: 10 }, // Start with 10%
      'user-service': { percentage: 5 },
      'analytics-service': { percentage: 20 } // Less critical first
    });
    
    // 3. Monitor and validate new services
    await this.monitorServiceHealth();
  }
  
  /**
   * Phase 2: Database Decomposition
   */
  async decomposeDatabase(): Promise<void> {
    // 1. Create service-specific databases
    const migrations = [
      this.createURLServiceDB(),
      this.createUserServiceDB(),
      this.createAnalyticsServiceDB(),
      this.createSubscriptionServiceDB()
    ];
    
    await Promise.all(migrations);
    
    // 2. Set up data synchronization
    await this.setupDataSynchronization();
    
    // 3. Gradually migrate data
    await this.migrateDataIncrementally();
  }
  
  /**
   * Phase 3: Event-Driven Integration
   */
  async enableEventDrivenArchitecture(): Promise<void> {
    // 1. Deploy Kafka cluster
    await this.deployKafka();
    
    // 2. Implement event publishers
    await this.setupEventPublishers();
    
    // 3. Implement event consumers
    await this.setupEventConsumers();
    
    // 4. Enable asynchronous communication
    await this.switchToAsyncCommunication();
  }
  
  private async setupTrafficSplitting(config: any): Promise<void> {
    // Implement intelligent traffic routing
    const gatewayConfig = {
      routes: [
        {
          path: '/api/urls/*',
          destinations: [
            { 
              service: 'monolith', 
              weight: 100 - config['url-service'].percentage 
            },
            { 
              service: 'url-service', 
              weight: config['url-service'].percentage 
            }
          ]
        }
      ]
    };
    
    await this.updateGatewayConfiguration(gatewayConfig);
  }
}
```

---

## 🎯 **SERVICE MESH INTEGRATION:**

### **Istio Configuration:**
```yaml
# istio/url-service-virtualservice.yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: url-service
spec:
  http:
  - match:
    - uri:
        prefix: "/api/urls"
    route:
    - destination:
        host: url-service
        port:
          number: 80
    fault:
      delay:          # Add latency for chaos engineering
        percentage:
          value: 0.1
        fixedDelay: 5s
    timeout: 10s
    retries:
      attempts: 3
      perTryTimeout: 3s

---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: url-service
spec:
  host: url-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        maxRequestsPerConnection: 10
    loadBalancer:
      simple: LEAST_CONN
    circuitBreaker:
      consecutiveErrors: 5
      interval: 10s
      baseEjectionTime: 30s
```

---

## ✅ **ACCEPTANCE CRITERIA:**
- [ ] URL Service deployed និង handling redirects independently
- [ ] User Service managing authentication separately
- [ ] Analytics Service processing events asynchronously  
- [ ] API Gateway routing requests correctly
- [ ] Inter-service communication via Kafka working
- [ ] Database per service implemented
- [ ] Zero-downtime migration completed
- [ ] Service mesh monitoring operational
- [ ] Circuit breakers preventing cascade failures
- [ ] Auto-scaling configuration functional

---

## 🧪 **TESTING CHECKLIST:**
- [ ] Service isolation testing
- [ ] Inter-service communication validation
- [ ] Event-driven workflow testing
- [ ] Database consistency verification
- [ ] Circuit breaker failure scenarios
- [ ] Load testing individual services
- [ ] End-to-end integration testing
- [ ] Migration rollback procedures validation

---

**Completion Date**: _________  
**Review By**: DevOps Lead + Architecture Team  
**Next Task**: Enterprise Security Implementation