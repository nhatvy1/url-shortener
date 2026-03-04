# TASK: Subscription System Core

**Ticket**: Monetization System  
**Priority**: P0 (Critical Revenue)  
**Assignee**: Senior Backend Developer + Payment Engineer  
**Estimate**: 5 days  
**Dependencies**: User Management System  

## 📋 TASK OVERVIEW

**Objective**: Implement complete subscription management system với Stripe integration  
**Success Criteria**: Users can subscribe, upgrade/downgrade plans, និង manage billing  

---

## 🎯 **SUBSCRIPTION REQUIREMENTS:**

### **Subscription Tiers:**
- [ ] **Free Tier**: 1,000 URLs/month, basic analytics
- [ ] **Pro Tier ($9/month)**: 10,000 URLs/month, advanced analytics, custom domains
- [ ] **Business Tier ($29/month)**: 50,000 URLs/month, team features, API access  
- [ ] **Enterprise Tier ($99/month)**: Unlimited URLs, white-label, SSO, priority support

### **Core Business Logic:**
- [ ] Subscription lifecycle management (create, update, cancel, reactivate)
- [ ] Usage tracking វាkvey real-time quotas enforcement
- [ ] Prorated billing for upgrades/downgrades
- [ ] Trial period handling (14 days free trial)
- [ ] Payment failure និង dunning management

### **Stripe Integration:**
- [ ] Customer និង subscription synchronization
- [ ] Webhook handling for payment events
- [ ] Invoice និង receipt generation
- [ ] Tax calculation și compliance

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **Subscription Database Schema:**
```sql
-- Subscription plans table
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    price_monthly DECIMAL(8,2) NOT NULL,
    price_yearly DECIMAL(8,2),
    stripe_price_id VARCHAR(255) NOT NULL,
    
    -- Features & Limits
    urls_per_month INTEGER NOT NULL DEFAULT 0, -- 0 = unlimited
    analytics_retention_days INTEGER DEFAULT 90,
    custom_domains_allowed INTEGER DEFAULT 0,
    api_calls_per_day INTEGER DEFAULT 0,
    team_members_max INTEGER DEFAULT 1,
    
    -- Feature flags
    features JSONB DEFAULT '{}',
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User subscriptions table
CREATE TABLE user_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID REFERENCES subscription_plans(id),
    
    -- Stripe data
    stripe_customer_id VARCHAR(255) NOT NULL,
    stripe_subscription_id VARCHAR(255) UNIQUE,
    
    -- Subscription details
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, canceled, past_due, incomplete
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    
    -- Billing
    billing_cycle VARCHAR(20) DEFAULT 'monthly', -- monthly, yearly
    next_billing_date TIMESTAMP WITH TIME ZONE,
    amount_due DECIMAL(8,2),
    currency VARCHAR(3) DEFAULT 'USD',
    
    -- Usage tracking
    urls_created_this_period INTEGER DEFAULT 0,
    api_calls_this_period INTEGER DEFAULT 0,
    period_reset_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    canceled_at TIMESTAMP WITH TIME ZONE,
    
    INDEX idx_user_subscriptions_user_id (user_id),
    INDEX idx_user_subscriptions_stripe_customer (stripe_customer_id),
    INDEX idx_user_subscriptions_status (status),
    INDEX idx_user_subscriptions_billing_date (next_billing_date)
);

-- Usage tracking table (for real-time quotas)
CREATE TABLE usage_tracking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES user_subscriptions(id),
    
    -- Usage metrics
    resource_type VARCHAR(50) NOT NULL, -- urls, api_calls, exports
    usage_count INTEGER NOT NULL DEFAULT 0,
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- Performance optimization
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, resource_type, period_start),
    INDEX idx_usage_tracking_user_period (user_id, period_start),
    INDEX idx_usage_tracking_resource (resource_type)
);

-- Payment history table
CREATE TABLE payment_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    subscription_id UUID REFERENCES user_subscriptions(id),
    
    -- Payment details
    stripe_payment_intent_id VARCHAR(255),
    stripe_invoice_id VARCHAR(255),
    amount DECIMAL(8,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL, -- succeeded, failed, pending
    
    -- Transaction details
    description TEXT,
    payment_method_type VARCHAR(50), -- card, bank_transfer, etc
    receipt_url TEXT,
    
    -- Timestamps
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_payment_history_user (user_id),
    INDEX idx_payment_history_subscription (subscription_id),
    INDEX idx_payment_history_status (status)
);
```

### **Subscription Service Implementation:**
```typescript
// src/services/SubscriptionService.ts
import Stripe from 'stripe';
import { UserSubscription, SubscriptionPlan, UsageTracking } from '../models';
import { EventEmitter } from 'events';

export class SubscriptionService extends EventEmitter {
  private stripe: Stripe;
  
  constructor() {
    super();
    this.stripe = new Stripe(process.env.STRIPE_SECRET_KEY!, {
      apiVersion: '2023-10-16'
    });
  }
  
  /**
   * Create new subscription for user
   */
  async createSubscription(data: {
    userId: string;
    planId: string;
    paymentMethodId: string;
    billingCycle: 'monthly' | 'yearly';
    trialDays?: number;
  }): Promise<UserSubscription> {
    
    try {
      // 1. Get subscription plan details
      const plan = await SubscriptionPlan.findById(data.planId);
      if (!plan || !plan.is_active) {
        throw new Error('Invalid subscription plan');
      }
      
      // 2. Get or create Stripe customer
      const customer = await this.getOrCreateStripeCustomer(data.userId);
      
      // 3. Attach payment method to customer
      await this.stripe.paymentMethods.attach(data.paymentMethodId, {
        customer: customer.id
      });
      
      // 4. Set as default payment method
      await this.stripe.customers.update(customer.id, {
        invoice_settings: {
          default_payment_method: data.paymentMethodId
        }
      });
      
      // 5. Create Stripe subscription
      const stripeSubscription = await this.stripe.subscriptions.create({
        customer: customer.id,
        items: [{
          price: plan.stripe_price_id
        }],
        payment_behavior: 'default_incomplete',
        payment_settings: { save_default_payment_method: 'on_subscription' },
        expand: ['latest_invoice.payment_intent'],
        trial_period_days: data.trialDays || (plan.name === 'pro' ? 14 : 0)
      });
      
      // 6. Create local subscription record
      const subscription = await UserSubscription.create({
        user_id: data.userId,
        plan_id: data.planId,
        stripe_customer_id: customer.id,
        stripe_subscription_id: stripeSubscription.id,
        status: stripeSubscription.status,
        current_period_start: new Date(stripeSubscription.current_period_start * 1000),
        current_period_end: new Date(stripeSubscription.current_period_end * 1000),
        trial_start: stripeSubscription.trial_start ? new Date(stripeSubscription.trial_start * 1000) : null,
        trial_end: stripeSubscription.trial_end ? new Date(stripeSubscription.trial_end * 1000) : null,
        billing_cycle: data.billingCycle,
        next_billing_date: new Date(stripeSubscription.current_period_end * 1000),
        currency: 'USD'
      });
      
      // 7. Initialize usage tracking
      await this.initializeUsageTracking(subscription.id, data.userId);
      
      // 8. Emit subscription event
      this.emit('subscription_created', {
        userId: data.userId,
        subscriptionId: subscription.id,
        planName: plan.name
      });
      
      return subscription;
      
    } catch (error) {
      console.error('Subscription creation error:', error);
      throw new Error('Failed to create subscription');
    }
  }
  
  /**
   * Handle Stripe webhook events
   */
  async handleWebhookEvent(event: Stripe.Event): Promise<void> {
    try {
      switch (event.type) {
        case 'customer.subscription.updated':
          await this.handleSubscriptionUpdated(event.data.object as Stripe.Subscription);
          break;
          
        case 'customer.subscription.deleted':
          await this.handleSubscriptionDeleted(event.data.object as Stripe.Subscription);
          break;
          
        case 'invoice.payment_succeeded':
          await this.handlePaymentSucceeded(event.data.object as Stripe.Invoice);
          break;
          
        case 'invoice.payment_failed':
          await this.handlePaymentFailed(event.data.object as Stripe.Invoice);
          break;
          
        case 'customer.subscription.trial_will_end':
          await this.handleTrialWillEnd(event.data.object as Stripe.Subscription);
          break;
          
        default:
          console.log(`Unhandled event type: ${event.type}`);
      }
    } catch (error) {
      console.error('Webhook handling error:', error);
      throw error;
    }
  }
  
  /**
   * Check and enforce usage limits
   */
  async checkUsageLimit(userId: string, resourceType: string): Promise<{
    allowed: boolean;
    current: number;
    limit: number;
    resetDate: Date;
  }> {
    
    // 1. Get user's current subscription
    const subscription = await UserSubscription.findOne({
      user_id: userId,
      status: 'active'
    }).populate('plan_id');
    
    if (!subscription) {
      // Free tier limits
      return await this.checkFreeTierLimits(userId, resourceType);
    }
    
    // 2. Get usage limits from plan
    const plan = subscription.plan_id as any;
    const limits = this.getResourceLimits(plan, resourceType);
    
    if (limits.unlimited) {
      return {
        allowed: true,
        current: 0,
        limit: -1, // Unlimited
        resetDate: subscription.current_period_end
      };
    }
    
    // 3. Get current usage for period
    const usage = await UsageTracking.findOne({
      user_id: userId,
      resource_type: resourceType,
      period_start: { $lte: new Date() },
      period_end: { $gte: new Date() }
    });
    
    const currentUsage = usage?.usage_count || 0;
    const allowed = currentUsage < limits.limit;
    
    return {
      allowed,
      current: currentUsage,
      limit: limits.limit,
      resetDate: subscription.current_period_end
    };
  }
  
  /**
   * Increment usage counter
   */
  async trackUsage(userId: string, resourceType: string, amount: number = 1): Promise<void> {
    const subscription = await UserSubscription.findOne({
      user_id: userId,
      status: 'active'
    });
    
    if (!subscription) {
      return; // Handle free tier separately
    }
    
    // Upsert usage tracking record
    await UsageTracking.findOneAndUpdate(
      {
        user_id: userId,
        resource_type: resourceType,
        period_start: subscription.current_period_start,
        period_end: subscription.current_period_end
      },
      {
        $inc: { usage_count: amount },
        $set: { last_updated: new Date() },
        $setOnInsert: {
          subscription_id: subscription.id,
          period_start: subscription.current_period_start,
          period_end: subscription.current_period_end
        }
      },
      { upsert: true }
    );
  }
  
  /**
   * Get or create Stripe customer
   */
  private async getOrCreateStripeCustomer(userId: string): Promise<Stripe.Customer> {
    const user = await User.findById(userId);
    if (!user) {
      throw new Error('User not found');
    }
    
    // Check if customer already exists
    const existing = await UserSubscription.findOne({
      user_id: userId,
      stripe_customer_id: { $exists: true }
    });
    
    if (existing?.stripe_customer_id) {
      return await this.stripe.customers.retrieve(existing.stripe_customer_id) as Stripe.Customer;
    }
    
    // Create new customer
    const customer = await this.stripe.customers.create({
      email: user.email,
      name: `${user.first_name} ${user.last_name}`.trim(),
      metadata: {
        userId: userId
      }
    });
    
    return customer;
  }
  
  /**
   * Handle subscription updated webhook
   */
  private async handleSubscriptionUpdated(stripeSubscription: Stripe.Subscription): Promise<void> {
    const subscription = await UserSubscription.findOne({
      stripe_subscription_id: stripeSubscription.id
    });
    
    if (!subscription) {
      console.warn('Subscription not found for Stripe ID:', stripeSubscription.id);
      return;
    }
    
    // Update subscription details
    await UserSubscription.updateOne(
      { id: subscription.id },
      {
        status: stripeSubscription.status,
        current_period_start: new Date(stripeSubscription.current_period_start * 1000),
        current_period_end: new Date(stripeSubscription.current_period_end * 1000),
        next_billing_date: new Date(stripeSubscription.current_period_end * 1000),
        updated_at: new Date()
      }
    );
    
    // Emit update event
    this.emit('subscription_updated', {
      subscriptionId: subscription.id,
      userId: subscription.user_id,
      status: stripeSubscription.status
    });
  }
  
  private getResourceLimits(plan: any, resourceType: string) {
    switch (resourceType) {
      case 'urls':
        return {
          limit: plan.urls_per_month,
          unlimited: plan.urls_per_month === 0
        };
      case 'api_calls':
        return {
          limit: plan.api_calls_per_day,
          unlimited: plan.api_calls_per_day === 0
        };
      default:
        return { limit: 0, unlimited: false };
    }
  }
}
```

### **Subscription Controller:**
```typescript
// src/controllers/SubscriptionController.ts
import { Request, Response, NextFunction } from 'express';
import { SubscriptionService } from '../services/SubscriptionService';
import { SubscriptionPlan } from '../models';

export class SubscriptionController {
  private subscriptionService = new SubscriptionService();
  
  /**
   * GET /api/plans
   * Get all available subscription plans
   */
  async getPlans(req: Request, res: Response, next: NextFunction) {
    try {
      const plans = await SubscriptionPlan.find({ is_active: true })
        .sort({ sort_order: 1 });
      
      res.json({
        success: true,
        data: plans
      });
    } catch (error) {
      next(error);
    }
  }
  
  /**
   * POST /api/subscribe
   * Create new subscription
   */
  async createSubscription(req: Request, res: Response, next: NextFunction) {
    try {
      const {
        planId,
        paymentMethodId,
        billingCycle = 'monthly'
      } = req.body;
      
      const subscription = await this.subscriptionService.createSubscription({
        userId: req.user.id,
        planId,
        paymentMethodId,
        billingCycle
      });
      
      res.status(201).json({
        success: true,
        data: subscription
      });
    } catch (error) {
      next(error);
    }
  }
  
  /**
   * GET /api/subscription
   * Get current user subscription
   */
  async getCurrentSubscription(req: Request, res: Response, next: NextFunction) {
    try {
      const subscription = await UserSubscription.findOne({
        user_id: req.user.id,
        status: { $in: ['active', 'trialing', 'past_due'] }
      }).populate('plan_id');
      
      if (!subscription) {
        return res.json({
          success: true,
          data: {
            plan: 'free',
            status: 'active'
          }
        });
      }
      
      // Get usage information
      const usage = await this.subscriptionService.getUsageSummary(req.user.id);
      
      res.json({
        success: true,
        data: {
          ...subscription.toJSON(),
          usage
        }
      });
    } catch (error) {
      next(error);
    }
  }
  
  /**
   * POST /api/webhooks/stripe
   * Handle Stripe webhook events
   */
  async handleStripeWebhook(req: Request, res: Response, next: NextFunction) {
    try {
      const sig = req.headers['stripe-signature'] as string;
      const endpointSecret = process.env.STRIPE_WEBHOOK_SECRET!;
      
      let event;
      
      try {
        event = stripe.webhooks.constructEvent(req.body, sig, endpointSecret);
      } catch (err) {
        console.error('Webhook signature verification failed:', err);
        return res.status(400).send('Webhook signature verification failed');
      }
      
      await this.subscriptionService.handleWebhookEvent(event);
      
      res.json({ received: true });
    } catch (error) {
      next(error);
    }
  }
}
```

---

## 💳 **STRIPE CONFIGURATION:**

### **Environment Variables:**
```bash
# Stripe configuration
STRIPE_PUBLISHABLE_KEY=pk_live_...
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Subscription settings
FREE_TIER_URL_LIMIT=1000
PRO_TIER_PRICE_MONTHLY=9.00
BUSINESS_TIER_PRICE_MONTHLY=29.00
ENTERPRISE_TIER_PRICE_MONTHLY=99.00
```

### **Stripe Products Setup:**
```javascript
// scripts/setup-stripe-products.js
const stripe = require('stripe')(process.env.STRIPE_SECRET_KEY);

async function setupSubscriptionPlans() {
  // Pro Plan
  const proProduct = await stripe.products.create({
    name: 'ShortLink Pro',
    description: 'Advanced URL shortening with analytics'
  });
  
  const proPrice = await stripe.prices.create({
    product: proProduct.id,
    unit_amount: 900, // $9.00
    currency: 'usd',
    recurring: { interval: 'month' }
  });
  
  // Business Plan
  const businessProduct = await stripe.products.create({
    name: 'ShortLink Business', 
    description: 'Team features and API access'
  });
  
  const businessPrice = await stripe.prices.create({
    product: businessProduct.id,
    unit_amount: 2900, // $29.00
    currency: 'usd',
    recurring: { interval: 'month' }
  });
}
```

---

## ✅ **ACCEPTANCE CRITERIA:**
- [ ] Complete subscription tiers implemented
- [ ] Stripe payment processing functional
- [ ] Usage tracking and quota enforcement working
- [ ] Webhook handling robust 
- [ ] Subscription lifecycle management complete
- [ ] Billing and invoicing operational
- [ ] Trial period handling correct
- [ ] Error handling comprehensive

---

## 🧪 **TESTING CHECKLIST:**
- [ ] Subscription creation flow testing
- [ ] Payment processing validation
- [ ] Usage limit enforcement testing
- [ ] Webhook event handling verification
- [ ] Trial period functionality testing
- [ ] Upgrade/downgrade flow testing
- [ ] Payment failure scenario testing
- [ ] Refund and cancellation testing

---

**Completion Date**: _________  
**Review By**: Senior Backend + Payment Engineer  
**Next Task**: Payment Dashboard UI