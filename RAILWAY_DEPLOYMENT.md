# Railway Deployment Guide for HybridLM API

## Prerequisites
- Railway account (https://railway.app)
- GitHub repository for the API
- API keys ready (OpenAI, Groq, Google OAuth)

## Step 1: Create Railway Project

1. Go to https://railway.app/new
2. Click "Deploy from GitHub repo"
3. Select your API repository
4. Railway will auto-detect the Dockerfile

## Step 2: Add Redis Database

1. In your Railway project, click "+ New"
2. Select "Database" → "Add Redis"
3. Redis will be automatically provisioned
4. Railway will create a `REDIS_URL` environment variable automatically

## Step 3: Configure Environment Variables

In Railway dashboard, go to your API service → "Variables" tab and add:

### Required Variables:

```bash
# API Keys
LLM_API_KEY=your_openai_api_key_here
GROQ_API_KEY=your_groq_api_key_here

# Google OAuth
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=https://your-api-domain.railway.app/api/v1/auth/callback

# CORS (Your Vercel frontend URL)
ALLOWED_ORIGINS=https://your-frontend.vercel.app,http://localhost:3000

# Server Config
PORT=8080
GIN_MODE=release

# Redis (automatically set by Railway, but verify)
REDIS_URL=${{Redis.REDIS_URL}}

# Optional: MongoDB (if using)
MONGODB_URI=your_mongodb_connection_string
```

### Optional Variables:

```bash
# Semantic Cache
SEMANTIC_CACHE_API_KEY=your_openai_key_or_separate_key

# Session Config
COOKIE_DOMAIN=.railway.app
COOKIE_SECURE=true
```

## Step 4: Reference Redis in Your Service

Railway automatically links services. The Redis URL is available as `${{Redis.REDIS_URL}}`.

If your code reads from `REDIS_URL` environment variable, it will automatically work.

## Step 5: Deploy

1. Railway will automatically deploy on every push to your main branch
2. Monitor deployment logs in Railway dashboard
3. Check health endpoint: `https://your-api.railway.app/api/v1/health`

## Step 6: Get Your API URL

After deployment:
1. Go to your service in Railway
2. Click "Settings" → "Networking"
3. Click "Generate Domain"
4. Your API will be available at: `https://your-service-name.up.railway.app`

## Step 7: Update Frontend Environment Variables (Vercel)

In your Vercel frontend project, update:

```bash
NEXT_PUBLIC_API_URL=https://your-api.up.railway.app
```

## Redis Connection

Railway provides Redis with this connection format:
```
redis://default:password@host:port
```

Your Go code should parse `REDIS_URL` correctly (it already does based on your config).

## Monitoring

- **Logs**: Railway Dashboard → Your Service → "Deployments" tab
- **Metrics**: Railway Dashboard → Your Service → "Metrics" tab
- **Health Check**: Railway automatically monitors `/api/v1/health`

## Cost Estimation

**Free Tier (Hobby Plan):**
- $5/month credit
- Redis: ~$1/month (256MB)
- API Service: ~$3-4/month (512MB RAM)
- **Total: ~$5/month (covered by free credit)**

**Paid Tier (if needed):**
- Redis (512MB): $2/month
- API Service (1GB RAM): $10/month

## Troubleshooting

### If deployment fails:

1. Check build logs in Railway dashboard
2. Verify Dockerfile builds locally:
   ```bash
   docker build -t hybridlm-api .
   docker run -p 8080:8080 hybridlm-api
   ```

### If Redis connection fails:

1. Verify `REDIS_URL` is set correctly
2. Check Railway service logs for connection errors
3. Ensure Redis service is running in Railway dashboard

### If health check fails:

1. Check if server is listening on `0.0.0.0:8080` (not `localhost`)
2. Verify `/api/v1/health` endpoint returns 200 OK
3. Check Railway logs for startup errors

## Rolling Back

If a deployment breaks:
1. Go to Railway Dashboard → Deployments
2. Find the last working deployment
3. Click "Redeploy"

## Custom Domain (Optional)

1. Railway Settings → Networking → Custom Domain
2. Add your domain: `api.yourdomain.com`
3. Update DNS records as instructed by Railway
4. Update `GOOGLE_REDIRECT_URL` and `ALLOWED_ORIGINS`

## Next Steps

1. Deploy API to Railway
2. Note your Railway API URL
3. Update Vercel frontend with new API URL
4. Test OAuth flow with new URLs
5. Monitor logs for any issues
