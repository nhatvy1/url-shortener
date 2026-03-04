# TASK: ML Infrastructure Platform

**Ticket**: ML Infrastructure  
**Priority**: P1 (High - AI Foundation)  
**Assignee**: ML Engineer + Data Engineer + DevOps  
**Estimate**: 6 days  
**Dependencies**: Microservices Architecture  

## 📋 TASK OVERVIEW

**Objective**: Build complete MLOps platform for AI-powered URL optimization  
**Success Criteria**: ML models deployed in production with automated training pipeline  

---

## 🎯 **ML INFRASTRUCTURE REQUIREMENTS:**

### **MLOps Platform:**
- [ ] **Model Development**: Jupyter Hub, MLflow, experiment tracking
- [ ] **Data Pipeline**: Apache Airflow, data versioning, feature stores
- [ ] **Model Training**: Scalable training infrastructure, GPU support
- [ ] **Model Serving**: Real-time inference API, A/B testing framework
- [ ] **Monitoring**: Model drift detection, performance monitoring
- [ ] **CI/CD**: Automated model deployment pipeline

### **Data Management:**
- [ ] **Data Lake**: S3-based data storage with partitioning
- [ ] **Feature Store**: Centralized feature management with Feast
- [ ] **Data Quality**: Automated data validation និង profiling
- [ ] **Real-time Streaming**: Kafka + Spark Streaming for live data
- [ ] **Data Governance**: Schema registry, data lineage tracking

### **Model Infrastructure:**
- [ ] **Training Cluster**: Kubernetes-based training jobs
- [ ] **Model Registry**: Centralized model versioning និង metadata
- [ ] **Inference Serving**: High-performance model serving with TensorFlow Serving
- [ ] **A/B Testing**: Traffic splitting for model experiments
- [ ] **Monitoring**: Model performance និង business metrics tracking

---

## 🛠 **TECHNICAL IMPLEMENTATION:**

### **MLOps Architecture:**
```yaml
# k8s/ml-platform/mlflow-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mlflow-server
  labels:
    app: mlflow-server
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mlflow-server
  template:
    metadata:
      labels:
        app: mlflow-server
    spec:
      containers:
      - name: mlflow-server
        image: shortlink/mlflow-server:latest
        ports:
        - containerPort: 5000
        env:
        - name: MLFLOW_BACKEND_STORE_URI
          value: "postgresql://mlflow:password@postgres-mlflow:5432/mlflow"
        - name: MLFLOW_DEFAULT_ARTIFACT_ROOT
          value: "s3://shortlink-mlflow-artifacts"
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aws-secrets
              key: access-key-id
        - name: AWS_SECRET_ACCESS_KEY  
          valueFrom:
            secretKeyRef:
              name: aws-secrets
              key: secret-access-key
        command:
          - mlflow
          - server
          - --host
          - 0.0.0.0
          - --port
          - "5000"
          - --backend-store-uri
          - $(MLFLOW_BACKEND_STORE_URI)
          - --default-artifact-root
          - $(MLFLOW_DEFAULT_ARTIFACT_ROOT)
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "2Gi"
            cpu: "1000m"

---
apiVersion: v1
kind: Service
metadata:
  name: mlflow-service
spec:
  selector:
    app: mlflow-server
  ports:
  - protocol: TCP
    port: 5000
    targetPort: 5000
  type: ClusterIP
```

### **Data Pipeline Implementation:**
```python
# airflow/dags/feature_pipeline.py
from airflow import DAG
from airflow.operators.python import PythonOperator
from airflow.operators.bash import BashOperator
from datetime import datetime, timedelta
import pandas as pd
import numpy as np
from feast import FeatureStore

default_args = {
    'owner': 'ml-team',
    'depends_on_past': False,
    'start_date': datetime(2026, 3, 1),
    'email_on_failure': True,
    'email_on_retry': False,
    'retries': 2,
    'retry_delay': timedelta(minutes=5)
}

dag = DAG(
    'feature_engineering_pipeline',
    default_args=default_args,
    description='Daily feature engineering for ML models',
    schedule_interval='@daily',
    catchup=False
)

def extract_click_features(**context):
    """Extract and transform click data into features"""
    from src.data.extractors import ClickDataExtractor
    from src.features.click_features import ClickFeatureEngineer
    
    # Get execution date
    execution_date = context['execution_date']
    start_date = execution_date
    end_date = execution_date + timedelta(days=1)
    
    # Extract raw click data
    extractor = ClickDataExtractor()
    raw_data = extractor.extract_clicks(start_date, end_date)
    
    # Engineer features
    feature_engineer = ClickFeatureEngineer()
    features = feature_engineer.transform(raw_data)
    
    # Save to feature store
    feature_store = FeatureStore(repo_path="/opt/feast/feature_repo")
    feature_store.write_to_offline_store("click_features", features)
    
    return f"Processed {len(features)} feature records"

def extract_url_features(**context):
    """Extract URL-based features"""
    from src.features.url_features import URLFeatureEngineer
    
    execution_date = context['execution_date']
    
    # Extract URL metadata and performance features
    feature_engineer = URLFeatureEngineer()
    features = feature_engineer.extract_daily_features(execution_date)
    
    # Save to feature store
    feature_store = FeatureStore(repo_path="/opt/feast/feature_repo")
    feature_store.write_to_offline_store("url_features", features)
    
    return f"Processed URL features for {execution_date}"

def train_ctr_model(**context):
    """Train click-through rate prediction model"""
    import mlflow
    import mlflow.sklearn
    from sklearn.ensemble import GradientBoostingRegressor
    from sklearn.model_selection import train_test_split
    from sklearn.metrics import mean_squared_error, r2_score
    
    execution_date = context['execution_date']
    
    # Set MLflow experiment
    mlflow.set_experiment("ctr_prediction")
    
    with mlflow.start_run(run_name=f"daily_training_{execution_date.strftime('%Y%m%d')}"):
        # Load features from feature store
        feature_store = FeatureStore(repo_path="/opt/feast/feature_repo")
        
        # Get training data (last 30 days)
        end_time = execution_date
        start_time = end_time - timedelta(days=30)
        
        features = feature_store.get_historical_features(
            entity_df=f"""
                SELECT url_id, timestamp
                FROM url_events
                WHERE timestamp BETWEEN '{start_time}' AND '{end_time}'
            """,
            features=[
                "click_features:total_clicks",
                "click_features:unique_visitors",
                "click_features:avg_session_duration",
                "url_features:url_length",
                "url_features:has_utm_params",
                "url_features:domain_authority"
            ]
        ).to_df()
        
        # Prepare training data
        X = features.drop(['url_id', 'timestamp', 'ctr'], axis=1)
        y = features['ctr']
        
        X_train, X_test, y_train, y_test = train_test_split(
            X, y, test_size=0.2, random_state=42
        )
        
        # Train model
        model = GradientBoostingRegressor(
            n_estimators=100,
            learning_rate=0.1,
            max_depth=6,
            random_state=42
        )
        
        model.fit(X_train, y_train)
        
        # Evaluate model  
        train_predictions = model.predict(X_train)
        test_predictions = model.predict(X_test)
        
        train_rmse = np.sqrt(mean_squared_error(y_train, train_predictions))
        test_rmse = np.sqrt(mean_squared_error(y_test, test_predictions))
        test_r2 = r2_score(y_test, test_predictions)
        
        # Log metrics
        mlflow.log_metric("train_rmse", train_rmse)
        mlflow.log_metric("test_rmse", test_rmse)
        mlflow.log_metric("test_r2", test_r2)
        mlflow.log_metric("training_samples", len(X_train))
        
        # Log model
        mlflow.sklearn.log_model(
            model, 
            "ctr_model",
            registered_model_name="ctr_prediction_model"
        )
        
        # Log feature importance
        feature_importance = pd.DataFrame({
            'feature': X.columns,
            'importance': model.feature_importances_
        }).sort_values('importance', ascending=False)
        
        mlflow.log_table(feature_importance, "feature_importance.json")
        
        return f"Model trained with R² score: {test_r2:.4f}"

def deploy_model_if_improved(**context):
    """Deploy model to production if performance improved"""
    import mlflow
    from mlflow.tracking import MlflowClient
    
    client = MlflowClient()
    
    # Get latest model from this run
    run_id = context['task_instance'].xcom_pull(task_ids='train_ctr_model')
    latest_model = client.get_latest_versions(
        "ctr_prediction_model", 
        stages=["None"]
    )[0]
    
    # Get current production model
    try:
        production_model = client.get_latest_versions(
            "ctr_prediction_model", 
            stages=["Production"]
        )[0]
        
        # Compare performance
        latest_metrics = client.get_run(latest_model.run_id).data.metrics
        production_metrics = client.get_run(production_model.run_id).data.metrics
        
        if latest_metrics['test_r2'] > production_metrics['test_r2']:
            # Promote new model to production
            client.transition_model_version_stage(
                name="ctr_prediction_model",
                version=latest_model.version,
                stage="Production"
            )
            
            # Archive old production model
            client.transition_model_version_stage(
                name="ctr_prediction_model", 
                version=production_model.version,
                stage="Archived"
            )
            
            return f"Deployed model version {latest_model.version} to production"
        else:
            return "Model performance not improved, keeping current production model"
            
    except IndexError:
        # No production model exists, deploy this one
        client.transition_model_version_stage(
            name="ctr_prediction_model",
            version=latest_model.version,
            stage="Production"
        )
        return f"Deployed first model version {latest_model.version} to production"

# Define task dependencies
extract_clicks = PythonOperator(
    task_id='extract_click_features',
    python_callable=extract_click_features,
    dag=dag
)

extract_urls = PythonOperator(
    task_id='extract_url_features', 
    python_callable=extract_url_features,
    dag=dag
)

train_model = PythonOperator(
    task_id='train_ctr_model',
    python_callable=train_ctr_model,
    dag=dag
)

deploy_model = PythonOperator(
    task_id='deploy_model_if_improved',
    python_callable=deploy_model_if_improved,
    dag=dag
)

# Set dependencies
[extract_clicks, extract_urls] >> train_model >> deploy_model
```

### **Feature Store Configuration:**
```python
# feast/feature_repo/feature_store.yaml
project: shortlink
registry:
  registry_store_type: sql
  path: postgresql://feast:password@postgres-feast:5432/feast
provider: aws
offline_store:
  type: redshift
  host: shortlink-redshift.cluster.amazonaws.com
  port: 5439
  database: shortlink_features
  user: feast_user
  s3_staging_location: s3://shortlink-feast-staging
online_store:
  type: redis
  connection_string: redis://redis-feast:6379

# feast/feature_repo/features.py
from feast import Entity, Feature, FeatureView, ValueType
from feast.data_source import RedshiftSource
from datetime import timedelta

# Define entities
url_entity = Entity(
    name="url_id",
    value_type=ValueType.STRING,
    description="Unique identifier for URLs"
)

user_entity = Entity(
    name="user_id", 
    value_type=ValueType.STRING,
    description="Unique identifier for users"
)

# Define data sources
click_features_source = RedshiftSource(
    name="click_features_source",
    query="""
        SELECT 
            url_id,
            user_id,
            total_clicks,
            unique_visitors,
            avg_session_duration,
            bounce_rate,
            mobile_click_ratio,
            event_timestamp
        FROM shortlink_features.click_features
    """,
    event_timestamp_column="event_timestamp",
    created_timestamp_column="created_timestamp"
)

url_features_source = RedshiftSource(
    name="url_features_source",
    query="""
        SELECT
            url_id,
            url_length,
            has_utm_params,
            domain_authority,
            is_custom_domain,
            created_hour,
            created_day_of_week,
            event_timestamp
        FROM shortlink_features.url_features  
    """,
    event_timestamp_column="event_timestamp",
    created_timestamp_column="created_timestamp"
)

# Define feature views
click_features_view = FeatureView(
    name="click_features",
    entities=["url_id", "user_id"],
    ttl=timedelta(days=90),
    features=[
        Feature(name="total_clicks", dtype=ValueType.INT64),
        Feature(name="unique_visitors", dtype=ValueType.INT64),
        Feature(name="avg_session_duration", dtype=ValueType.FLOAT),
        Feature(name="bounce_rate", dtype=ValueType.FLOAT),
        Feature(name="mobile_click_ratio", dtype=ValueType.FLOAT)
    ],
    batch_source=click_features_source
)

url_features_view = FeatureView(
    name="url_features",
    entities=["url_id"],
    ttl=timedelta(days=365),
    features=[
        Feature(name="url_length", dtype=ValueType.INT64),
        Feature(name="has_utm_params", dtype=ValueType.BOOL),
        Feature(name="domain_authority", dtype=ValueType.FLOAT),
        Feature(name="is_custom_domain", dtype=ValueType.BOOL),
        Feature(name="created_hour", dtype=ValueType.INT64),
        Feature(name="created_day_of_week", dtype=ValueType.INT64)
    ],
    batch_source=url_features_source
)
```

### **Real-time Model Serving:**
```python
# ml_serving/ctr_prediction_api.py
from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel
import mlflow.pyfunc
import numpy as np
import pandas as pd
from feast import FeatureStore
import logging
import time

app = FastAPI(title="CTR Prediction API", version="1.0.0")

# Load model on startup
model = None
feature_store = None

class CTRPredictionRequest(BaseModel):
    url_id: str
    user_id: str = None
    features: dict = None

class CTRPredictionResponse(BaseModel):
    url_id: str
    predicted_ctr: float
    confidence: float
    model_version: str
    prediction_timestamp: str

@app.on_event("startup")
async def load_model():
    global model, feature_store
    
    try:
        # Load production model
        model_name = "ctr_prediction_model"
        model_stage = "Production"
        model = mlflow.pyfunc.load_model(f"models:/{model_name}/{model_stage}")
        
        # Initialize feature store
        feature_store = FeatureStore(repo_path="/opt/feast/feature_repo")
        
        logging.info(f"Loaded model {model_name} from {model_stage} stage")
        
    except Exception as e:
        logging.error(f"Failed to load model: {e}")
        raise

@app.post("/predict", response_model=CTRPredictionResponse)
async def predict_ctr(
    request: CTRPredictionRequest,
    background_tasks: BackgroundTasks
):
    """Predict click-through rate for a URL"""
    
    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    try:
        start_time = time.time()
        
        # Get features from feature store or use provided features
        if request.features:
            features = pd.DataFrame([request.features])
        else:
            features = await get_features_from_store(request.url_id, request.user_id)
        
        # Make prediction
        prediction = model.predict(features)[0]
        
        # Calculate confidence (simplified measure)
        confidence = min(0.95, max(0.5, 1.0 - abs(0.5 - prediction) * 2))
        
        prediction_time = time.time() - start_time
        
        response = CTRPredictionResponse(
            url_id=request.url_id,
            predicted_ctr=float(prediction),
            confidence=float(confidence),
            model_version="1.0.0",
            prediction_timestamp=pd.Timestamp.now().isoformat()
        )
        
        # Log prediction metrics asynchronously
        background_tasks.add_task(
            log_prediction_metrics,
            request.url_id,
            prediction,
            confidence,
            prediction_time
        )
        
        return response
        
    except Exception as e:
        logging.error(f"Prediction error: {e}")
        raise HTTPException(status_code=500, detail="Prediction failed")

async def get_features_from_store(url_id: str, user_id: str = None) -> pd.DataFrame:
    """Retrieve features from feature store"""
    
    # Create entity DataFrame
    entity_df = pd.DataFrame({
        'url_id': [url_id],
        'event_timestamp': [pd.Timestamp.now()]
    })
    
    if user_id:
        entity_df['user_id'] = user_id
    
    # Get online features
    feature_vector = feature_store.get_online_features(
        features=[
            'url_features:url_length',
            'url_features:has_utm_params', 
            'url_features:domain_authority',
            'url_features:is_custom_domain',
            'click_features:total_clicks',
            'click_features:unique_visitors'
        ],
        entity_rows=entity_df.to_dict('records')
    ).to_df()
    
    # Remove metadata columns
    features = feature_vector.drop(['url_id', 'user_id'], axis=1, errors='ignore')
    
    return features

async def log_prediction_metrics(
    url_id: str, 
    prediction: float, 
    confidence: float, 
    response_time: float
):
    """Log prediction metrics for monitoring"""
    
    metrics = {
        'url_id': url_id,
        'predicted_ctr': prediction,
        'confidence': confidence,
        'response_time_ms': response_time * 1000,
        'timestamp': pd.Timestamp.now().isoformat()
    }
    
    # Log to monitoring system
    logging.info(f"Prediction metrics: {metrics}")

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    
    return {
        "status": "healthy",
        "model_loaded": model is not None,
        "feature_store_connected": feature_store is not None,
        "timestamp": pd.Timestamp.now().isoformat()
    }

@app.get("/metrics")
async def get_metrics():
    """Prometheus metrics endpoint"""
    
    # Return metrics in Prometheus format
    return """
# HELP ctr_prediction_requests_total Total prediction requests
# TYPE ctr_prediction_requests_total counter
ctr_prediction_requests_total 1234

# HELP ctr_prediction_response_time Response time in seconds
# TYPE ctr_prediction_response_time histogram
ctr_prediction_response_time_bucket{le="0.1"} 950
ctr_prediction_response_time_bucket{le="0.5"} 1200
ctr_prediction_response_time_bucket{le="1.0"} 1230
"""
```

---

## 🔍 **MODEL MONITORING & DRIFT DETECTION:**

### **Data Drift Detection:**
```python
# monitoring/drift_detection.py
import pandas as pd
import numpy as np
from scipy import stats
from typing import Dict, List, Tuple
import logging

class DriftDetector:
    
    def __init__(self, reference_data: pd.DataFrame, threshold: float = 0.05):
        self.reference_data = reference_data
        self.threshold = threshold
    
    def detect_drift(self, current_data: pd.DataFrame) -> Dict[str, any]:
        """Detect data drift using statistical tests"""
        
        drift_results = {
            'has_drift': False,
            'drift_features': [],
            'drift_scores': {},
            'summary': {}
        }
        
        # Check each feature for drift
        for feature in self.reference_data.columns:
            if feature in current_data.columns:
                drift_score, p_value = self.kolmogorov_smirnov_test(
                    self.reference_data[feature],
                    current_data[feature]
                )
                
                drift_results['drift_scores'][feature] = {
                    'ks_statistic': drift_score,
                    'p_value': p_value,
                    'has_drift': p_value < self.threshold
                }
                
                if p_value < self.threshold:
                    drift_results['has_drift'] = True
                    drift_results['drift_features'].append(feature)
        
        # Calculate summary statistics
        drift_results['summary'] = {
            'total_features': len(self.reference_data.columns),
            'drifted_features': len(drift_results['drift_features']),
            'drift_percentage': len(drift_results['drift_features']) / len(self.reference_data.columns) * 100
        }
        
        return drift_results
    
    def kolmogorov_smirnov_test(self, reference: pd.Series, current: pd.Series) -> Tuple[float, float]:
        """Perform Kolmogorov-Smirnov test for distribution comparison"""
        
        # Remove NaN values
        ref_clean = reference.dropna()
        cur_clean = current.dropna()
        
        # Perform KS test
        ks_statistic, p_value = stats.ks_2samp(ref_clean, cur_clean)
        
        return ks_statistic, p_value

class ModelPerformanceMonitor:
    
    def __init__(self, model_name: str):
        self.model_name = model_name
    
    async def monitor_predictions(self, predictions: pd.DataFrame) -> Dict[str, any]:
        """Monitor model predictions for anomalies"""
        
        monitoring_results = {
            'prediction_stats': self.calculate_prediction_stats(predictions),
            'anomalies': self.detect_prediction_anomalies(predictions),
            'performance_metrics': await self.calculate_performance_metrics(predictions)
        }
        
        return monitoring_results
    
    def calculate_prediction_stats(self, predictions: pd.DataFrame) -> Dict[str, float]:
        """Calculate basic statistics for predictions"""
        
        return {
            'mean_prediction': predictions['predicted_ctr'].mean(),
            'std_prediction': predictions['predicted_ctr'].std(),
            'min_prediction': predictions['predicted_ctr'].min(),
            'max_prediction': predictions['predicted_ctr'].max(),
            'prediction_count': len(predictions)
        }
    
    def detect_prediction_anomalies(self, predictions: pd.DataFrame) -> List[Dict]:
        """Detect anomalous predictions"""
        
        anomalies = []
        
        # Z-score based anomaly detection
        z_scores = np.abs(stats.zscore(predictions['predicted_ctr']))
        anomaly_threshold = 3.0
        
        anomaly_indices = np.where(z_scores > anomaly_threshold)[0]
        
        for idx in anomaly_indices:
            anomalies.append({
                'url_id': predictions.iloc[idx]['url_id'],
                'predicted_ctr': predictions.iloc[idx]['predicted_ctr'],
                'z_score': z_scores[idx],
                'timestamp': predictions.iloc[idx]['timestamp']
            })
        
        return anomalies
```

---

## ✅ **ACCEPTANCE CRITERIA:**
- [ ] MLflow tracking server operational
- [ ] Automated feature engineering pipeline running daily
- [ ] Feature store serving online និង offline features  
- [ ] Model training pipeline with automated deployment
- [ ] Real-time model inference API (< 100ms response)
- [ ] Model performance monitoring dashboard
- [ ] Data drift detection alerts functional
- [ ] A/B testing framework for model experiments
- [ ] Scalable training infrastructure on Kubernetes
- [ ] Model registry with versioning និង metadata

---

## 🧪 **TESTING CHECKLIST:**
- [ ] ML pipeline end-to-end testing
- [ ] Model training និង deployment automation
- [ ] Feature store performance testing
- [ ] Model serving API load testing
- [ ] Data drift detection algorithm validation
- [ ] Model monitoring dashboard functionality
- [ ] A/B testing framework integration
- [ ] Rollback procedure for model deployments

---

**Completion Date**: _________  
**Review By**: ML Engineer + Data Team Lead  
**Next Task**: Predictive Analytics Implementation