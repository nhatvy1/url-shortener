# TASK: Advanced Analytics Dashboard

**Ticket**: Advanced Analytics  
**Priority**: P1 (High - Revenue Driver)  
**Assignee**: Full-Stack Developer + Data Analyst  
**Estimate**: 4 days  
**Dependencies**: Basic Analytics System  

## 📋 TASK OVERVIEW

**Objective**: Build comprehensive analytics dashboard with insights និង data visualization  
**Success Criteria**: Users can analyze URL performance with actionable insights  

---

## 🎯 **ANALYTICS REQUIREMENTS:**

### **Dashboard Metrics:**
- [ ] **Performance Overview**: CTR, total clicks, unique visitors, conversion rates
- [ ] **Geographic Analysis**: Click distribution by country/region with world map
- [ ] **Time-Based Analytics**: Hourly, daily, weekly, monthly trends
- [ ] **Device & Browser Analytics**: Mobile vs desktop, browser breakdown
- [ ] **Referrer Analysis**: Traffic sources, social media performance
- [ ] **Link Performance Comparison**: Top performing vs underperforming URLs

### **Advanced Features:**
- [ ] **Custom Date Ranges**: Flexible date filtering with presets
- [ ] **Real-Time Updates**: Live click tracking with WebSocket updates
- [ ] **Data Export**: CSV, PDF, Excel export capabilities
- [ ] **Automated Insights**: AI-powered recommendations និង anomaly detection
- [ ] **Custom Metrics**: User-defined KPIs និង goals
- [ ] **Cohort Analysis**: User behavior tracking over time

### **Business Intelligence:**
- [ ] **Revenue Attribution**: Link performance correlation with conversions
- [ ] **A/B Testing Results**: Performance comparison between link variations  
- [ ] **Audience Segmentation**: User behavior patterns និង demographics
- [ ] **Predictive Analytics**: Future performance predictions

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **Analytics Database Schema:**
```sql
-- Enhanced analytics tables for advanced reporting
CREATE TABLE url_analytics_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_url_id UUID REFERENCES short_urls(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    
    -- Date dimension
    date_key DATE NOT NULL,
    hour_key INTEGER NOT NULL CHECK (hour_key BETWEEN 0 AND 23),
    
    -- Metrics (pre-aggregated for performance)
    total_clicks INTEGER DEFAULT 0,
    unique_clicks INTEGER DEFAULT 0,
    bounce_rate DECIMAL(5,2) DEFAULT 0,
    avg_session_duration INTEGER DEFAULT 0, -- seconds
    
    -- Geographic breakdowns
    top_countries JSONB DEFAULT '{}',
    top_regions JSONB DEFAULT '{}',
    
    -- Device breakdowns  
    mobile_clicks INTEGER DEFAULT 0,
    desktop_clicks INTEGER DEFAULT 0,
    tablet_clicks INTEGER DEFAULT 0,
    
    -- Browser/OS breakdowns
    browser_stats JSONB DEFAULT '{}',
    os_stats JSONB DEFAULT '{}',
    
    -- Referrer analysis
    referrer_stats JSONB DEFAULT '{}',
    social_media_clicks INTEGER DEFAULT 0,
    direct_clicks INTEGER DEFAULT 0,
    
    -- Performance metrics
    avg_response_time DECIMAL(8,2) DEFAULT 0, -- milliseconds
    error_rate DECIMAL(5,4) DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(short_url_id, date_key, hour_key),
    INDEX idx_analytics_summary_url_date (short_url_id, date_key),
    INDEX idx_analytics_summary_user_date (user_id, date_key),
    INDEX idx_analytics_summary_date (date_key)
);

-- Real-time analytics cache
CREATE TABLE analytics_realtime (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    short_url_id UUID REFERENCES short_urls(id) ON DELETE CASCADE,
    
    -- Real-time metrics (last 24 hours)
    clicks_last_hour INTEGER DEFAULT 0,
    clicks_last_24h INTEGER DEFAULT 0,
    unique_visitors_24h INTEGER DEFAULT 0,
    
    -- Geographic data (JSON for flexibility)
    live_countries JSONB DEFAULT '{}',
    live_referrers JSONB DEFAULT '{}',
    
    -- Performance data
    avg_response_time_24h DECIMAL(8,2) DEFAULT 0,
    
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(short_url_id),
    INDEX idx_analytics_realtime_updated (last_updated)
);

-- User analytics preferences
CREATE TABLE user_analytics_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    
    -- Dashboard preferences
    default_date_range VARCHAR(20) DEFAULT 'last_30_days',
    favorite_metrics TEXT[] DEFAULT ARRAY['total_clicks', 'unique_clicks', 'ctr'],
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Notification preferences
    enable_email_reports BOOLEAN DEFAULT TRUE,
    report_frequency VARCHAR(20) DEFAULT 'weekly', -- daily, weekly, monthly
    threshold_alerts JSONB DEFAULT '{}',
    
    -- Export preferences
    export_format VARCHAR(10) DEFAULT 'csv', -- csv, pdf, excel
    include_raw_data BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Analytics insights (AI-generated)
CREATE TABLE analytics_insights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    short_url_id UUID REFERENCES short_urls(id) ON DELETE CASCADE,
    
    -- Insight details
    insight_type VARCHAR(50) NOT NULL, -- trend, anomaly, recommendation, prediction
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    confidence_score DECIMAL(3,2) DEFAULT 0, -- 0.00 to 1.00
    
    -- Insight data
    metrics JSONB DEFAULT '{}',
    recommendations JSONB DEFAULT '{}',
    
    -- Metadata
    priority VARCHAR(20) DEFAULT 'medium', -- low, medium, high, critical
    status VARCHAR(20) DEFAULT 'active', -- active, dismissed, archived
    
    -- Analytics
    viewed_at TIMESTAMP WITH TIME ZONE,
    dismissed_at TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    
    INDEX idx_analytics_insights_user (user_id),
    INDEX idx_analytics_insights_type (insight_type),
    INDEX idx_analytics_insights_priority (priority)
);
```

### **Analytics Service Implementation:**
```typescript
// src/services/AnalyticsService.ts
import { URLAnalyticsSummary, AnalyticsRealtime, AnalyticsInsights } from '../models';
import { Cache } from '../utils/Cache';
import { EventEmitter } from 'events';

export class AnalyticsService extends EventEmitter {
  private cache = new Cache();
  
  /**
   * Get comprehensive dashboard analytics
   */
  async getDashboardAnalytics(userId: string, options: {
    dateRange: string;
    urlId?: string;
    timezone?: string;
  }): Promise<DashboardAnalytics> {
    
    const { startDate, endDate } = this.parseDateRange(options.dateRange);
    
    // Build query filters
    const filters: any = {
      user_id: userId,
      date_key: { $gte: startDate, $lte: endDate }
    };
    
    if (options.urlId) {
      filters.short_url_id = options.urlId;
    }
    
    // Execute parallel queries for performance
    const [
      overview,
      timeSeriesData,
      geographicData,
      deviceData,
      referrerData,
      topPerformingUrls,
      insights
    ] = await Promise.all([
      this.getMetricsOverview(filters),
      this.getTimeSeriesData(filters, options.dateRange),
      this.getGeographicAnalytics(filters),
      this.getDeviceAnalytics(filters),
      this.getReferrerAnalytics(filters),
      this.getTopPerformingUrls(userId, startDate, endDate),
      this.getActiveInsights(userId, options.urlId)
    ]);
    
    return {
      overview,
      timeSeriesData,
      geographicData,
      deviceData,
      referrerData,
      topPerformingUrls,
      insights,
      metadata: {
        dateRange: options.dateRange,
        generatedAt: new Date(),
        timezone: options.timezone || 'UTC'
      }
    };
  }
  
  /**
   * Get real-time analytics (live updates)
   */
  async getRealtimeAnalytics(userId: string, urlId?: string): Promise<RealtimeAnalytics> {
    const filters: any = { user_id: userId };
    if (urlId) filters.short_url_id = urlId;
    
    // Get real-time data (cached for 30 seconds)
    const cacheKey = `realtime_analytics:${userId}:${urlId || 'all'}`;
    let realtimeData = await this.cache.get(cacheKey);
    
    if (!realtimeData) {
      realtimeData = await this.calculateRealtimeMetrics(filters);
      await this.cache.set(cacheKey, realtimeData, 30); // 30 second cache
    }
    
    return realtimeData;
  }
  
  /**
   * Generate AI-powered insights
   */
  async generateInsights(userId: string, urlId?: string): Promise<AnalyticsInsight[]> {
    // Get historical data for analysis
    const historicalData = await this.getHistoricalData(userId, urlId, 90); // 90 days
    
    const insights: AnalyticsInsight[] = [];
    
    // 1. Trend Analysis
    const trendInsight = await this.analyzeTrends(historicalData);
    if (trendInsight) insights.push(trendInsight);
    
    // 2. Anomaly Detection
    const anomalies = await this.detectAnomalies(historicalData);
    insights.push(...anomalies);
    
    // 3. Performance Recommendations
    const recommendations = await this.generateRecommendations(historicalData);
    insights.push(...recommendations);
    
    // 4. Predictive Analytics
    const predictions = await this.generatePredictions(historicalData);
    if (predictions) insights.push(predictions);
    
    // Store insights in database
    for (const insight of insights) {
      await AnalyticsInsights.create({
        user_id: userId,
        short_url_id: urlId,
        insight_type: insight.type,
        title: insight.title,
        description: insight.description,
        confidence_score: insight.confidence,
        metrics: insight.metrics,
        recommendations: insight.recommendations,
        priority: insight.priority,
        expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000) // 7 days
      });
    }
    
    return insights;
  }
  
  /**
   * Export analytics data
   */
  async exportAnalytics(userId: string, options: {
    format: 'csv' | 'pdf' | 'excel';
    dateRange: string;
    includeRawData?: boolean;
    urlIds?: string[];
  }): Promise<{ downloadUrl: string; fileName: string }> {
    
    const { startDate, endDate } = this.parseDateRange(options.dateRange);
    
    // Get comprehensive data
    const data = await this.getExportData(userId, {
      startDate,
      endDate,
      includeRawData: options.includeRawData,
      urlIds: options.urlIds
    });
    
    // Generate export file
    let fileBuffer: Buffer;
    let fileName: string;
    let contentType: string;
    
    switch (options.format) {
      case 'csv':
        fileBuffer = await this.generateCSVExport(data);
        fileName = `analytics_${Date.now()}.csv`;
        contentType = 'text/csv';
        break;
        
      case 'pdf':
        fileBuffer = await this.generatePDFExport(data);
        fileName = `analytics_report_${Date.now()}.pdf`;
        contentType = 'application/pdf';
        break;
        
      case 'excel':
        fileBuffer = await this.generateExcelExport(data);
        fileName = `analytics_${Date.now()}.xlsx`;
        contentType = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
        break;
        
      default:
        throw new Error('Unsupported export format');
    }
    
    // Upload to S3 or file storage
    const downloadUrl = await this.uploadExportFile(fileBuffer, fileName, contentType);
    
    return { downloadUrl, fileName };
  }
  
  /**
   * Calculate metrics overview
   */
  private async getMetricsOverview(filters: any): Promise<MetricsOverview> {
    const aggregation = await URLAnalyticsSummary.aggregate([
      { $match: filters },
      {
        $group: {
          _id: null,
          totalClicks: { $sum: '$total_clicks' },
          uniqueClicks: { $sum: '$unique_clicks' },
          totalUrls: { $addToSet: '$short_url_id' },
          avgResponseTime: { $avg: '$avg_response_time' },
          errorRate: { $avg: '$error_rate' }
        }
      }
    ]);
    
    const result = aggregation[0] || {
      totalClicks: 0,
      uniqueClicks: 0,
      totalUrls: [],
      avgResponseTime: 0,
      errorRate: 0
    };
    
    // Calculate derived metrics
    const ctr = result.totalClicks > 0 ? (result.uniqueClicks / result.totalClicks) * 100 : 0;
    const totalUrls = result.totalUrls.length;
    
    return {
      totalClicks: result.totalClicks,
      uniqueClicks: result.uniqueClicks,
      clickThroughRate: ctr,
      totalUrls: totalUrls,
      avgClicksPerUrl: totalUrls > 0 ? result.totalClicks / totalUrls : 0,
      avgResponseTime: result.avgResponseTime,
      errorRate: result.errorRate * 100 // Convert to percentage
    };
  }
  
  /**
   * Analyze trends and generate insights
   */
  private async analyzeTrends(data: any[]): Promise<AnalyticsInsight | null> {
    if (data.length < 7) return null; // Need at least 7 days of data
    
    // Calculate week-over-week growth
    const recentWeek = data.slice(-7);
    const previousWeek = data.slice(-14, -7);
    
    const recentTotal = recentWeek.reduce((sum, day) => sum + day.total_clicks, 0);
    const previousTotal = previousWeek.reduce((sum, day) => sum + day.total_clicks, 0);
    
    if (previousTotal === 0) return null;
    
    const growthRate = ((recentTotal - previousTotal) / previousTotal) * 100;
    
    let insight: AnalyticsInsight | null = null;
    
    if (Math.abs(growthRate) > 20) { // Significant change
      insight = {
        type: 'trend',
        title: growthRate > 0 ? 'Strong Growth Detected' : 'Declining Performance Alert',
        description: `Your links have ${growthRate > 0 ? 'grown' : 'declined'} by ${Math.abs(growthRate).toFixed(1)}% compared to last week.`,
        confidence: 0.85,
        priority: Math.abs(growthRate) > 50 ? 'high' : 'medium',
        metrics: {
          growthRate: growthRate,
          recentClicks: recentTotal,
          previousClicks: previousTotal
        },
        recommendations: growthRate > 0 
          ? ['Continue your current strategy', 'Consider scaling successful campaigns']
          : ['Review recent changes', 'Analyze underperforming content', 'Check for technical issues']
      };
    }
    
    return insight;
  }
  
  private parseDateRange(range: string): { startDate: Date; endDate: Date } {
    const now = new Date();
    const endDate = new Date(now);
    let startDate: Date;
    
    switch (range) {
      case 'today':
        startDate = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        break;
      case 'yesterday':
        startDate = new Date(now.getTime() - 24 * 60 * 60 * 1000);
        endDate.setTime(startDate.getTime());
        break;
      case 'last_7_days':
        startDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
        break;
      case 'last_30_days':
        startDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
        break;
      case 'last_90_days':
        startDate = new Date(now.getTime() - 90 * 24 * 60 * 60 * 1000);
        break;
      default:
        startDate = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
    }
    
    return { startDate, endDate };
  }
}

// Type definitions
interface DashboardAnalytics {
  overview: MetricsOverview;
  timeSeriesData: TimeSeriesData[];
  geographicData: GeographicData;
  deviceData: DeviceData;
  referrerData: ReferrerData;
  topPerformingUrls: URLPerformance[];
  insights: AnalyticsInsight[];
  metadata: {
    dateRange: string;
    generatedAt: Date;
    timezone: string;
  };
}

interface MetricsOverview {
  totalClicks: number;
  uniqueClicks: number;
  clickThroughRate: number;
  totalUrls: number;
  avgClicksPerUrl: number;
  avgResponseTime: number;
  errorRate: number;
}

interface AnalyticsInsight {
  type: string;
  title: string;
  description: string;
  confidence: number;
  priority: string;
  metrics: any;
  recommendations: string[];
}
```

### **Analytics Dashboard Component:**
```tsx
// src/components/AnalyticsDashboard.tsx
import React, { useState, useEffect } from 'react';
import { Line, Bar, Pie, WorldMap } from 'react-chartjs-2';
import { DateRangePicker } from './DateRangePicker';
import { MetricsCard } from './MetricsCard';
import { InsightsPanel } from './InsightsPanel';

export const AnalyticsDashboard: React.FC = () => {
  const [analytics, setAnalytics] = useState<DashboardAnalytics | null>(null);
  const [dateRange, setDateRange] = useState('last_30_days');
  const [selectedUrl, setSelectedUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  
  useEffect(() => {
    loadAnalytics();
  }, [dateRange, selectedUrl]);
  
  const loadAnalytics = async () => {
    setLoading(true);
    try {
      const response = await apiClient.get('/api/analytics/dashboard', {
        params: { dateRange, urlId: selectedUrl }
      });
      setAnalytics(response.data.data);
    } catch (error) {
      console.error('Failed to load analytics:', error);
    } finally {
      setLoading(false);
    }
  };
  
  if (loading) {
    return <div className="analytics-loading">Loading analytics...</div>;
  }
  
  if (!analytics) {
    return <div className="analytics-error">Failed to load analytics data</div>;
  }
  
  return (
    <div className="analytics-dashboard">
      {/* Header Controls */}
      <div className="dashboard-header">
        <h1>Analytics Dashboard</h1>
        <div className="controls">
          <DateRangePicker
            value={dateRange}
            onChange={setDateRange}
          />
          <URLSelector
            value={selectedUrl}
            onChange={setSelectedUrl}
          />
        </div>
      </div>
      
      {/* Key Metrics Overview */}
      <div className="metrics-overview">
        <MetricsCard
          title="Total Clicks"
          value={analytics.overview.totalClicks.toLocaleString()}
          change="+12.5%"
          trend="up"
        />
        <MetricsCard
          title="Unique Visitors"
          value={analytics.overview.uniqueClicks.toLocaleString()}
          change="+8.3%"
          trend="up"
        />
        <MetricsCard
          title="Click Rate"
          value={`${analytics.overview.clickThroughRate.toFixed(1)}%`}
          change="-2.1%"
          trend="down"
        />
        <MetricsCard
          title="Avg Response Time"
          value={`${analytics.overview.avgResponseTime.toFixed(0)}ms`}
          change="-15ms"
          trend="up"
        />
      </div>
      
      {/* Charts Section */}
      <div className="charts-grid">
        {/* Time Series Chart */}
        <div className="chart-container">
          <h3>Clicks Over Time</h3>
          <Line
            data={{
              labels: analytics.timeSeriesData.map(d => d.date),
              datasets: [{
                label: 'Total Clicks',
                data: analytics.timeSeriesData.map(d => d.clicks),
                borderColor: 'rgb(59, 130, 246)',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                tension: 0.4
              }]
            }}
            options={{
              responsive: true,
              maintainAspectRatio: false,
              scales: {
                y: {
                  beginAtZero: true
                }
              }
            }}
          />
        </div>
        
        {/* Geographic Distribution */}
        <div className="chart-container">
          <h3>Geographic Distribution</h3>
          <WorldMap
            data={analytics.geographicData}
            colorScale={['#f0f9ff', '#0369a1']}
          />
        </div>
        
        {/* Device Breakdown */}
        <div className="chart-container">
          <h3>Device Types</h3>
          <Pie
            data={{
              labels: ['Mobile', 'Desktop', 'Tablet'],
              datasets: [{
                data: [
                  analytics.deviceData.mobile,
                  analytics.deviceData.desktop,
                  analytics.deviceData.tablet
                ],
                backgroundColor: ['#10b981', '#3b82f6', '#f59e0b']
              }]
            }}
          />
        </div>
        
        {/* Top Referrers */}
        <div className="chart-container">
          <h3>Top Referrers</h3>
          <Bar
            data={{
              labels: analytics.referrerData.top.map(r => r.domain),
              datasets: [{
                label: 'Clicks',
                data: analytics.referrerData.top.map(r => r.clicks),
                backgroundColor: 'rgba(59, 130, 246, 0.8)'
              }]
            }}
            options={{
              indexAxis: 'y' as const,
              responsive: true,
              maintainAspectRatio: false
            }}
          />
        </div>
      </div>
      
      {/* Insights Panel */}
      <InsightsPanel insights={analytics.insights} />
      
      {/* Top Performing URLs */}
      <div className="top-urls-section">
        <h3>Top Performing URLs</h3>
        <div className="urls-table">
          {analytics.topPerformingUrls.map((url) => (
            <div key={url.id} className="url-row">
              <div className="url-info">
                <a href={url.shortUrl} target="_blank" rel="noopener noreferrer">
                  {url.shortCode}
                </a>
                <span className="original-url">{url.originalUrl}</span>
              </div>
              <div className="url-metrics">
                <span>{url.clicks} clicks</span>
                <span>{url.ctr}% CTR</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
```

---

## 📊 **VISUALIZATION COMPONENTS:**

### **Interactive Charts Configuration:**
```typescript
// Chart.js configuration with custom styling
export const chartDefaults = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: 'top' as const,
    },
    tooltip: {
      mode: 'index' as const,
      intersect: false,
    },
  },
  scales: {
    x: {
      display: true,
      title: {
        display: true,
        text: 'Date'
      }
    },
    y: {
      display: true,
      title: {
        display: true,
        text: 'Clicks'
      },
      beginAtZero: true
    }
  },
  elements: {
    point: {
      radius: 3,
      hoverRadius: 6
    }
  }
};
```

---

## ✅ **ACCEPTANCE CRITERIA:**
- [ ] Comprehensive analytics dashboard functional
- [ ] Real-time updates working via WebSocket
- [ ] Data visualization charts responsive និង interactive
- [ ] Export functionality (CSV, PDF, Excel) operational
- [ ] AI insights generation working correctly
- [ ] Geographic analytics with world map display
- [ ] Device និង browser breakdowns accurate
- [ ] Performance metrics tracking < 100ms query response
- [ ] Custom date ranges working properly
- [ ] Mobile-responsive dashboard design

---

## 🧪 **TESTING CHECKLIST:**
- [ ] Dashboard loading performance testing
- [ ] Chart rendering with large datasets
- [ ] Real-time updates functionality verification
- [ ] Export generation testing (all formats)
- [ ] Mobile responsiveness validation
- [ ] Data accuracy verification
- [ ] Insight generation algorithm testing
- [ ] User preferences persistence testing

---

**Completion Date**: _________  
**Review By**: Full-Stack Developer + Data Analyst  
**Next Task**: API Marketplace Development