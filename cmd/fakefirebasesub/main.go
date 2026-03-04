package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com.br/lucas-mezencio/pdsi1/internal/config"
	"github.com.br/lucas-mezencio/pdsi1/internal/infrastructure/scheduler"
)

const (
	defaultAPIBaseURL = "http://localhost:8080/api/v1"
)

type createUserResponse struct {
	ID string `json:"id"`
}

type createDoctorResponse struct {
	ID string `json:"id"`
}

type createPrescriptionResponse struct {
	ID string `json:"id"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	appConfig, err := config.Load()
	if err != nil {
		log.Printf("config load failed: %v", err)
		return
	}

	baseURL := envString("API_BASE_URL", defaultAPIBaseURL)
	if err := waitForAPI(ctx, baseURL+"/health"); err != nil {
		log.Printf("api not ready: %v", err)
		return
	}

	userPayload := map[string]any{
		"name":           "Test User",
		"email":          fmt.Sprintf("user-%s@example.com", uuid.New().String()),
		"phone":          "+1234567890",
		"firebase_token": "fake-token",
	}
	userBytes, err := doJSON(ctx, http.MethodPost, baseURL+"/users", userPayload)
	if err != nil {
		log.Printf("create user failed: %v", err)
		return
	}
	var userResp createUserResponse
	if err := json.Unmarshal(userBytes, &userResp); err != nil {
		log.Printf("decode user response failed: %v", err)
		return
	}
	printJSON("Created user", userBytes)

	doctorPayload := map[string]any{
		"name":           "Dr. Test",
		"email":          fmt.Sprintf("doctor-%s@example.com", uuid.New().String()),
		"phone":          "+1098765432",
		"license_number": "LIC-12345",
		"specialty":      "Testing",
	}
	doctorBytes, err := doJSON(ctx, http.MethodPost, baseURL+"/doctors", doctorPayload)
	if err != nil {
		log.Printf("create doctor failed: %v", err)
		return
	}
	var doctorResp createDoctorResponse
	if err := json.Unmarshal(doctorBytes, &doctorResp); err != nil {
		log.Printf("decode doctor response failed: %v", err)
		return
	}
	printJSON("Created doctor", doctorBytes)

	start := time.Now().Add(3 * time.Second).Truncate(time.Second)
	startTime := start.Format("15:04:05")
	startSecond := time.Now().Add(10 * time.Second).Truncate(time.Second)
	startSecondTime := startSecond.Format("15:04:05")

	prescriptionPayload := map[string]any{
		"user_id":  userResp.ID,
		"medic_id": doctorResp.ID,
		"medicaments": []map[string]any{
			{
				"name":      "SecondsMed",
				"dosage":    "1",
				"frequency": "00:00:01",
				"time":      []string{startTime},
				"doses":     10,
			},
		},
	}
	prescriptionBytes, err := doJSON(ctx, http.MethodPost, baseURL+"/prescriptions", prescriptionPayload)
	if err != nil {
		log.Printf("create prescription failed: %v", err)
		return
	}
	var prescriptionResp createPrescriptionResponse
	if err := json.Unmarshal(prescriptionBytes, &prescriptionResp); err != nil {
		log.Printf("decode prescription response failed: %v", err)
		return
	}
	printJSON("Created prescription", prescriptionBytes)

	prescriptionSecondPayload := map[string]any{
		"user_id":  userResp.ID,
		"medic_id": doctorResp.ID,
		"medicaments": []map[string]any{
			{
				"name":      "FiveSecondMed",
				"dosage":    "1",
				"frequency": "00:00:05",
				"time":      []string{startSecondTime},
				"doses":     12,
			},
		},
	}
	prescriptionSecondBytes, err := doJSON(ctx, http.MethodPost, baseURL+"/prescriptions", prescriptionSecondPayload)
	if err != nil {
		log.Printf("create second prescription failed: %v", err)
		return
	}
	var prescriptionSecondResp createPrescriptionResponse
	if err := json.Unmarshal(prescriptionSecondBytes, &prescriptionSecondResp); err != nil {
		log.Printf("decode second prescription response failed: %v", err)
		return
	}
	printJSON("Created prescription", prescriptionSecondBytes)

	listBytes, err := doJSON(ctx, http.MethodGet, baseURL+"/prescriptions?user_id="+userResp.ID, nil)
	if err != nil {
		log.Printf("list prescriptions failed: %v", err)
		return
	}
	printJSON("Prescriptions for user", listBytes)

	fmt.Printf("Expected notification times (1s): %s\n", expectedTimes(start, 10))
	fmt.Printf("Expected notification times (5s): %s\n", expectedTimes(startSecond, 12, 5*time.Second))

	redisClient := redis.NewClient(&redis.Options{Addr: appConfig.RedisAddr})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("redis connect failed: %v", err)
		return
	}
	defer redisClient.Close()

	logger := watermill.NopLogger{}
	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        redisClient,
		ConsumerGroup: "fakefirebasesub-" + uuid.New().String(),
		Consumer:      "cli",
		BlockTime:     100 * time.Millisecond,
		OldestId:      "$",
	}, &logger)
	if err != nil {
		log.Printf("subscriber init failed: %v", err)
		return
	}
	defer subscriber.Close()

	messages, err := subscriber.Subscribe(ctx, scheduler.NotificationTopic)
	if err != nil {
		log.Printf("subscribe failed: %v", err)
		return
	}

	fmt.Println("Listening for notifications...")
	for i := 0; i < 22; i++ {
		select {
		case <-ctx.Done():
			log.Printf("timed out waiting for notifications: %v", ctx.Err())
			return
		case msg := <-messages:
			var job scheduler.NotificationJob
			if err := json.Unmarshal(msg.Payload, &job); err != nil {
				msg.Nack()
				log.Printf("decode notification job failed: %v", err)
				return
			}
			fmt.Printf("fakefirebasesub: %s %s %s at %s\n", job.UserID, job.MedicamentName, job.Dosage, job.ScheduledAt.Format(time.RFC3339))
			msg.Ack()
		}
	}
}

func doJSON(ctx context.Context, method, url string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %s", string(bytes.TrimSpace(data)))
	}

	return data, nil
}

func printJSON(label string, payload []byte) {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, payload, "", "  "); err != nil {
		fmt.Printf("%s: %s\n", label, string(payload))
		return
	}
	fmt.Printf("%s:\n%s\n", label, pretty.String())
}

func waitForAPI(ctx context.Context, url string) error {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

func expectedTimes(start time.Time, count int, interval ...time.Duration) string {
	if count <= 0 {
		return "[]"
	}
	step := time.Second
	if len(interval) > 0 && interval[0] > 0 {
		step = interval[0]
	}
	values := make([]string, 0, count)
	for i := 0; i < count; i++ {
		values = append(values, start.Add(time.Duration(i)*step).Format(time.RFC3339))
	}
	return fmt.Sprintf("[%s]", joinStrings(values, ", "))
}

func joinStrings(values []string, sep string) string {
	if len(values) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for i, value := range values {
		if i > 0 {
			buf.WriteString(sep)
		}
		buf.WriteString(value)
	}
	return buf.String()
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
