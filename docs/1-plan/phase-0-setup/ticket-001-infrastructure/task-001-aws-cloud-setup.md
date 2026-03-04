# TASK: AWS Cloud Infrastructure Setup

**Ticket**: Infrastructure Setup  
**Priority**: P0 (Critical)  
**Assignee**: DevOps Engineer  
**Estimate**: 2 days  
**Dependencies**: None  

## 📋 TASK OVERVIEW

**Objective**: Setup complete AWS cloud infrastructure for ShortLink platform  
**Success Criteria**: All AWS services configured và production-ready  

---

## 🎯 **REQUIREMENTS:**

### **AWS Account & Organization:**
- [ ] Setup AWS Organization với billing consolidation
- [ ] Create production và staging environments
- [ ] Configure IAM roles និង policies
- [ ] Setup CloudTrail for compliance auditing

### **Networking Infrastructure:**
- [ ] VPC setup với public/private subnets
- [ ] Internet Gateway និង NAT Gateway configuration
- [ ] Security Groups និង NACLs
- [ ] Route53 DNS configuration

### **Core Services Configuration:**
- [ ] ALB (Application Load Balancer) setup
- [ ] Auto Scaling Groups configuration
- [ ] CloudFront CDN setup
- [ ] CloudWatch monitoring និង alerting

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **VPC & Networking Setup:**
```bash
#!/bin/bash
# AWS VPC Infrastructure Setup Script

# Create VPC
aws ec2 create-vpc \
  --cidr-block 10.0.0.0/16 \
  --tag-specifications 'ResourceType=vpc,Tags=[{Key=Name,Value=shortlink-vpc},{Key=Environment,Value=production}]'

# Create Internet Gateway
aws ec2 create-internet-gateway \
  --tag-specifications 'ResourceType=internet-gateway,Tags=[{Key=Name,Value=shortlink-igw}]'

# Create Public Subnets (Multi-AZ)
aws ec2 create-subnet \
  --vpc-id $VPC_ID \
  --cidr-block 10.0.1.0/24 \
  --availability-zone us-east-1a \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=shortlink-public-1a}]'

aws ec2 create-subnet \
  --vpc-id $VPC_ID \
  --cidr-block 10.0.2.0/24 \
  --availability-zone us-east-1b \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=shortlink-public-1b}]'

# Create Private Subnets
aws ec2 create-subnet \
  --vpc-id $VPC_ID \
  --cidr-block 10.0.10.0/24 \
  --availability-zone us-east-1a \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=shortlink-private-1a}]'

aws ec2 create-subnet \
  --vpc-id $VPC_ID \
  --cidr-block 10.0.11.0/24 \
  --availability-zone us-east-1b \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=shortlink-private-1b}]'
```

### **Terraform Infrastructure Code:**
```hcl
# terraform/infrastructure/vpc.tf
resource "aws_vpc" "shortlink_vpc" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name        = "shortlink-vpc"
    Environment = var.environment
    Project     = "shortlink"
  }
}

# Application Load Balancer
resource "aws_lb" "shortlink_alb" {
  name               = "shortlink-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets           = aws_subnet.public[*].id

  enable_deletion_protection = var.environment == "production" ? true : false

  tags = {
    Name        = "shortlink-alb"
    Environment = var.environment
  }
}

# CloudFront Distribution
resource "aws_cloudfront_distribution" "shortlink_cdn" {
  origin {
    domain_name = aws_lb.shortlink_alb.dns_name
    origin_id   = "ShortLink-ALB"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"

  aliases = ["shortlink.com", "www.shortlink.com"]

  default_cache_behavior {
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "ShortLink-ALB"
    compress              = true
    viewer_protocol_policy = "redirect-to-https"

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn = aws_acm_certificate.shortlink_cert.arn
    ssl_support_method  = "sni-only"
  }

  tags = {
    Name        = "shortlink-cdn"
    Environment = var.environment
  }
}
```

---

## 🔐 **IAM ROLES & POLICIES:**

### **Application IAM Role:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "rds:DescribeDBInstances",
        "elasticache:DescribeCacheClusters",
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ],
      "Resource": "arn:aws:*:*:*:shortlink-*"
    },
    {
      "Effect": "Allow", 
      "Action": [
        "cloudwatch:PutMetricData",
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "*"
    }
  ]
}
```

---

## ✅ **ACCEPTANCE CRITERIA:**
- [ ] VPC និង subnets configured correctly
- [ ] Load balancer operational với health checks
- [ ] CloudFront CDN serving static content
- [ ] Route53 DNS resolution working
- [ ] Security groups restrictive by default
- [ ] All resources tagged properly
- [ ] CloudWatch monitoring active
- [ ] Cost optimization measures applied

---

## 🧪 **TESTING CHECKLIST:**
- [ ] Network connectivity test between subnets
- [ ] Load balancer health check verification
- [ ] CDN cache hit ratio validation
- [ ] DNS resolution testing from multiple locations
- [ ] Security group rules validation
- [ ] SSL certificate validation

---

**Completion Date**: _________  
**Review By**: DevOps Lead  
**Next Task**: Database Setup