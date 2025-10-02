// PesXChange Frontend API Configuration
// Update your frontend to use these new API endpoints

export const API_CONFIG = {
  // Base URL - Update this to your deployed backend URL
  BASE_URL: process.env.NODE_ENV === 'production' 
    ? 'https://your-pesxchange-backend.onrender.com'
    : 'http://localhost:8080',
  
  // API Endpoints
  ENDPOINTS: {
    // Authentication
    AUTH: {
      PESU_LOGIN: '/api/auth/pesu',
      CHECK_SRN: '/api/auth/check-srn',
    },
    
    // User/Profile Management
    PROFILE: {
      GET: '/api/profile/:id',
      UPDATE: '/api/profile/:id',
    },
    
    // Items
    ITEMS: {
      LIST: '/api/items',
      GET: '/api/items/:id',
      CREATE: '/api/items',
      UPDATE: '/api/items/:id',
      DELETE: '/api/items/:id',
    },
    
    // Messaging
    MESSAGES: {
      SEND: '/api/messages',
      GET: '/api/messages',
      MARK_READ: '/api/messages/read',
      ACTIVE_CHATS: '/api/active-chats',
    },
    
    // Health Check
    HEALTH: '/health',
  }
};

// API Client Helper Functions
export class PesXChangeAPI {
  constructor(baseURL = API_CONFIG.BASE_URL) {
    this.baseURL = baseURL;
    this.token = null;
  }

  // Set authentication token
  setToken(token) {
    this.token = token;
  }

  // Get authentication headers
  getHeaders(includeAuth = false) {
    const headers = {
      'Content-Type': 'application/json',
    };
    
    if (includeAuth && this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }
    
    return headers;
  }

  // Generic request method
  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;
    const config = {
      ...options,
      headers: {
        ...this.getHeaders(options.requireAuth),
        ...options.headers,
      },
    };

    try {
      const response = await fetch(url, config);
      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.error || `HTTP error! status: ${response.status}`);
      }
      
      return data;
    } catch (error) {
      console.error('API Request failed:', error);
      throw error;
    }
  }

  // Authentication Methods
  async loginWithPESU(username, password) {
    return this.request(API_CONFIG.ENDPOINTS.AUTH.PESU_LOGIN, {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  }

  async checkSRN(srn) {
    return this.request(`${API_CONFIG.ENDPOINTS.AUTH.CHECK_SRN}?srn=${srn}`);
  }

  // Profile Methods
  async getProfile(userId) {
    return this.request(API_CONFIG.ENDPOINTS.PROFILE.GET.replace(':id', userId));
  }

  async updateProfile(userId, updates) {
    return this.request(API_CONFIG.ENDPOINTS.PROFILE.UPDATE.replace(':id', userId), {
      method: 'PUT',
      body: JSON.stringify(updates),
      requireAuth: true,
    });
  }

  // Item Methods
  async getItems(filters = {}) {
    const queryParams = new URLSearchParams(filters).toString();
    const endpoint = queryParams ? 
      `${API_CONFIG.ENDPOINTS.ITEMS.LIST}?${queryParams}` : 
      API_CONFIG.ENDPOINTS.ITEMS.LIST;
    
    return this.request(endpoint);
  }

  async getItem(itemId) {
    return this.request(API_CONFIG.ENDPOINTS.ITEMS.GET.replace(':id', itemId));
  }

  async createItem(itemData) {
    return this.request(API_CONFIG.ENDPOINTS.ITEMS.CREATE, {
      method: 'POST',
      body: JSON.stringify(itemData),
      requireAuth: true,
    });
  }

  async updateItem(itemId, updates) {
    return this.request(API_CONFIG.ENDPOINTS.ITEMS.UPDATE.replace(':id', itemId), {
      method: 'PUT',
      body: JSON.stringify(updates),
      requireAuth: true,
    });
  }

  async deleteItem(itemId) {
    return this.request(API_CONFIG.ENDPOINTS.ITEMS.DELETE.replace(':id', itemId), {
      method: 'DELETE',
      requireAuth: true,
    });
  }

  // Message Methods
  async sendMessage(receiverId, itemId, content) {
    return this.request(API_CONFIG.ENDPOINTS.MESSAGES.SEND, {
      method: 'POST',
      body: JSON.stringify({
        receiver_id: receiverId,
        item_id: itemId,
        content,
      }),
      requireAuth: true,
    });
  }

  async getMessages(otherUserId, itemId, limit = 50, offset = 0) {
    const params = new URLSearchParams({
      other_user_id: otherUserId,
      item_id: itemId,
      limit: limit.toString(),
      offset: offset.toString(),
    });
    
    return this.request(`${API_CONFIG.ENDPOINTS.MESSAGES.GET}?${params}`, {
      requireAuth: true,
    });
  }

  async markMessagesAsRead(otherUserId, itemId) {
    return this.request(API_CONFIG.ENDPOINTS.MESSAGES.MARK_READ, {
      method: 'PUT',
      body: JSON.stringify({
        other_user_id: otherUserId,
        item_id: itemId,
      }),
      requireAuth: true,
    });
  }

  async getActiveChats() {
    return this.request(API_CONFIG.ENDPOINTS.MESSAGES.ACTIVE_CHATS, {
      requireAuth: true,
    });
  }

  // Health Check
  async healthCheck() {
    return this.request(API_CONFIG.ENDPOINTS.HEALTH);
  }
}

// Example usage:
/*
const api = new PesXChangeAPI();

// Login
const loginResult = await api.loginWithPESU('PES2UG21CS123', 'password');
if (loginResult.success) {
  api.setToken(loginResult.data.access_token);
  
  // Now you can make authenticated requests
  const items = await api.getItems({ search: 'laptop' });
  const newItem = await api.createItem({
    title: 'Gaming Laptop',
    description: 'High-performance gaming laptop',
    price: 50000,
    location: 'Bangalore',
    condition: 'like-new',
    categories: ['Electronics', 'Computers']
  });
}
*/

export default PesXChangeAPI;