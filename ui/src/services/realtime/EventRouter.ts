/**
 * EventRouter.ts
 * Routes real-time events to appropriate handlers with pattern matching
 */

import { v4 as uuidv4 } from 'uuid';

export type EventHandler<T = any> = (data: T) => void;

export interface SubscriptionOptions {
  priority?: number;
  filter?: (data: any) => boolean;
  transform?: (data: any) => any;
}

export interface Subscription {
  id: string;
  pattern: string;
  handler: EventHandler;
  options?: SubscriptionOptions;
}

export interface RealtimeEvent {
  id: string;
  type: string;
  data: any;
  timestamp: Date;
  source: 'websocket' | 'sse';
  metadata?: Record<string, any>;
}

/**
 * Ring buffer for event history
 */
class RingBuffer<T> {
  private buffer: T[];
  private size: number;
  private index: number = 0;

  constructor(size: number) {
    this.size = size;
    this.buffer = new Array(size);
  }

  push(item: T): void {
    this.buffer[this.index] = item;
    this.index = (this.index + 1) % this.size;
  }

  getAll(): T[] {
    const result: T[] = [];
    let currentIndex = this.index;
    
    for (let i = 0; i < this.size; i++) {
      const item = this.buffer[currentIndex];
      if (item !== undefined) {
        result.push(item);
      }
      currentIndex = (currentIndex + 1) % this.size;
    }
    
    return result;
  }

  clear(): void {
    this.buffer = new Array(this.size);
    this.index = 0;
  }
}

export class EventRouter {
  private subscriptions: Map<string, Set<Subscription>> = new Map();
  private eventBuffer: RingBuffer<RealtimeEvent> = new RingBuffer(1000);
  private patternCache: Map<string, RegExp> = new Map();

  /**
   * Subscribe to events with pattern matching
   * Patterns support wildcards: * matches single segment, ** matches multiple segments
   * Examples:
   * - "platform.*.status" matches "platform.prod.status"
   * - "platform.**" matches all platform events
   * - "**.error" matches all error events
   */
  subscribe<T>(pattern: string, handler: EventHandler<T>, options?: SubscriptionOptions): () => void {
    const subscription: Subscription = {
      id: uuidv4(),
      pattern,
      handler,
      options,
    };

    // Get or create subscription set for this pattern
    let patternSubs = this.subscriptions.get(pattern);
    if (!patternSubs) {
      patternSubs = new Set();
      this.subscriptions.set(pattern, patternSubs);
    }

    patternSubs.add(subscription);

    // Return unsubscribe function
    return () => this.unsubscribe(subscription.id);
  }

  /**
   * Unsubscribe from events
   */
  private unsubscribe(subscriptionId: string): void {
    for (const [pattern, subs] of this.subscriptions.entries()) {
      const toRemove = Array.from(subs).find(sub => sub.id === subscriptionId);
      if (toRemove) {
        subs.delete(toRemove);
        if (subs.size === 0) {
          this.subscriptions.delete(pattern);
        }
        break;
      }
    }
  }

  /**
   * Route incoming event to subscribers
   */
  routeEvent(event: RealtimeEvent): void {
    const matchingSubscriptions = this.findMatchingSubscriptions(event.type);

    // Sort by priority if specified
    const sorted = matchingSubscriptions.sort((a, b) => {
      const priorityA = a.options?.priority || 0;
      const priorityB = b.options?.priority || 0;
      return priorityB - priorityA;
    });

    sorted.forEach(sub => {
      try {
        // Apply filter if specified
        if (sub.options?.filter && !sub.options.filter(event.data)) {
          return;
        }

        // Apply transformation if specified
        const data = sub.options?.transform
          ? sub.options.transform(event.data)
          : event.data;

        // Call handler
        sub.handler(data);
      } catch (error) {
        console.error(`Event handler error for pattern ${sub.pattern}:`, error);
      }
    });

    // Buffer event for replay
    this.eventBuffer.push(event);
  }

  /**
   * Find subscriptions matching the event type
   */
  private findMatchingSubscriptions(eventType: string): Subscription[] {
    const matching: Subscription[] = [];

    for (const [pattern, subs] of this.subscriptions.entries()) {
      if (this.matchesPattern(eventType, pattern)) {
        matching.push(...Array.from(subs));
      }
    }

    return matching;
  }

  /**
   * Check if event type matches pattern
   */
  private matchesPattern(eventType: string, pattern: string): boolean {
    // Exact match
    if (eventType === pattern) {
      return true;
    }

    // Get or create regex for pattern
    let regex = this.patternCache.get(pattern);
    if (!regex) {
      // Convert pattern to regex
      const regexStr = pattern
        .split('.')
        .map(segment => {
          if (segment === '**') {
            return '.*'; // Match any number of segments
          } else if (segment === '*') {
            return '[^.]+'; // Match single segment
          } else {
            return segment.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); // Escape special chars
          }
        })
        .join('\\.');
      
      regex = new RegExp(`^${regexStr}$`);
      this.patternCache.set(pattern, regex);
    }

    return regex.test(eventType);
  }

  /**
   * Replay buffered events matching pattern
   */
  replayEvents(pattern: string, handler: EventHandler): void {
    const events = this.eventBuffer.getAll();
    
    events.forEach(event => {
      if (this.matchesPattern(event.type, pattern)) {
        try {
          handler(event.data);
        } catch (error) {
          console.error('Error replaying event:', error);
        }
      }
    });
  }

  /**
   * Get event history matching pattern
   */
  getEventHistory(pattern: string): RealtimeEvent[] {
    const events = this.eventBuffer.getAll();
    return events.filter(event => this.matchesPattern(event.type, pattern));
  }

  /**
   * Clear all subscriptions
   */
  clearSubscriptions(): void {
    this.subscriptions.clear();
  }

  /**
   * Clear event buffer
   */
  clearEventBuffer(): void {
    this.eventBuffer.clear();
  }

  /**
   * Get subscription statistics
   */
  getStats(): {
    totalSubscriptions: number;
    patternCount: number;
    bufferedEvents: number;
  } {
    let totalSubscriptions = 0;
    for (const subs of this.subscriptions.values()) {
      totalSubscriptions += subs.size;
    }

    return {
      totalSubscriptions,
      patternCount: this.subscriptions.size,
      bufferedEvents: this.eventBuffer.getAll().length,
    };
  }
}
