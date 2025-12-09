package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

// NNTPClient represents an NNTP client connection
type NNTPClient struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

// DialNNTP connects to an NNTP server
func DialNNTP(host string, port int, useSSL bool) (*NNTPClient, error) {
	address := fmt.Sprintf("%s:%d", host, port)

	var conn net.Conn
	var err error

	if useSSL {
		conn, err = tls.Dial("tcp", address, &tls.Config{
			InsecureSkipVerify: true, // In production, verify certificates properly
		})
	} else {
		conn, err = net.Dial("tcp", address)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client := &NNTPClient{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}

	// Read welcome message
	_, _, err = client.readResponse()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read welcome: %w", err)
	}

	return client, nil
}

// Close closes the connection
func (c *NNTPClient) Close() error {
	c.sendCommand("QUIT")
	return c.conn.Close()
}

// Authenticate logs in to the server
func (c *NNTPClient) Authenticate(username, password string) error {
	// Send username
	if err := c.sendCommand(fmt.Sprintf("AUTHINFO USER %s", username)); err != nil {
		return err
	}

	code, _, err := c.readResponse()
	if err != nil {
		return err
	}

	if code != 381 { // 381 = Password required
		return fmt.Errorf("unexpected response to USER command: %d", code)
	}

	// Send password
	if err := c.sendCommand(fmt.Sprintf("AUTHINFO PASS %s", password)); err != nil {
		return err
	}

	code, _, err = c.readResponse()
	if err != nil {
		return err
	}

	if code != 281 { // 281 = Authentication successful
		return fmt.Errorf("authentication failed: %d", code)
	}

	return nil
}

// GetArticle retrieves an article by message ID
func (c *NNTPClient) GetArticle(messageID string) ([]byte, error) {
	// Clean message ID
	messageID = strings.Trim(messageID, "<>")

	if err := c.sendCommand(fmt.Sprintf("ARTICLE <%s>", messageID)); err != nil {
		return nil, err
	}

	code, _, err := c.readResponse()
	if err != nil {
		return nil, err
	}

	if code != 220 { // 220 = Article follows
		return nil, fmt.Errorf("article not found: %d", code)
	}

	// Read article body until "." on a line by itself
	var body []byte
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		// Check for termination
		if len(line) == 3 && line[0] == '.' && line[1] == '\r' && line[2] == '\n' {
			break
		}

		// Remove dot-stuffing (lines starting with ".." become ".")
		if len(line) > 1 && line[0] == '.' && line[1] == '.' {
			line = line[1:]
		}

		body = append(body, line...)
	}

	return body, nil
}

// SelectGroup selects a newsgroup
func (c *NNTPClient) SelectGroup(group string) error {
	if err := c.sendCommand(fmt.Sprintf("GROUP %s", group)); err != nil {
		return err
	}

	code, _, err := c.readResponse()
	if err != nil {
		return err
	}

	if code != 211 { // 211 = Group selected
		return fmt.Errorf("failed to select group: %d", code)
	}

	return nil
}

// sendCommand sends a command to the server
func (c *NNTPClient) sendCommand(cmd string) error {
	if _, err := c.writer.WriteString(cmd + "\r\n"); err != nil {
		return err
	}
	return c.writer.Flush()
}

// readResponse reads and parses a response from the server
func (c *NNTPClient) readResponse() (int, string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return 0, "", err
	}

	line = strings.TrimSpace(line)

	// Parse response code
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 1 {
		return 0, "", fmt.Errorf("invalid response: %s", line)
	}

	code, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid response code: %s", parts[0])
	}

	message := ""
	if len(parts) > 1 {
		message = parts[1]
	}

	return code, message, nil
}

// TestConnection tests if we can connect and authenticate
func TestNNTPConnection(host string, port int, username, password string, useSSL bool) error {
	client, err := DialNNTP(host, port, useSSL)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Authenticate(username, password); err != nil {
		return err
	}

	return nil
}

// DownloadSegment downloads a single segment and writes it to the writer
func (c *NNTPClient) DownloadSegment(messageID string, w io.Writer) error {
	data, err := c.GetArticle(messageID)
	if err != nil {
		return err
	}

	// In a real implementation, you would decode yEnc here
	// For now, just write the raw data
	_, err = w.Write(data)
	return err
}
