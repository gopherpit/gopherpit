// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package email

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"net"
	"net/mail"
	"strings"
	"testing"
)

type SMTPRecorder struct {
	Port    int
	Message *SMTPMessage
}

func NewSMTPRecorder(t *testing.T) (*SMTPRecorder, error) {
	l, err := net.Listen("tcp", "")
	if err != nil {
		return nil, err
	}

	recorder := &SMTPRecorder{
		Port: l.Addr().(*net.TCPAddr).Port,
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				t.Fatal(err)
			}
			go func(conn net.Conn) {
				defer conn.Close()

				reader := bufio.NewReader(conn)
				writer := bufio.NewWriter(conn)

				writer.WriteString("220 Welcome\r\n")
				writer.Flush()

				s, err := reader.ReadString('\n')
				if err != nil {
					t.Fatal(err)
				}
				t.Log(strings.TrimSpace(s))

				writer.WriteString("250 Hello\r\n")
				writer.Flush()

				s, err = reader.ReadString('\n')
				if err != nil {
					t.Fatal(err)
				}
				t.Log(strings.TrimSpace(s))

				writer.WriteString("250 Sender\r\n")
				writer.Flush()

				s, err = reader.ReadString('\n')
				if err != nil {
					t.Fatal(err)
				}
				t.Log(strings.TrimSpace(s))

				for {
					writer.WriteString("250 Recipient\r\n")
					writer.Flush()

					s, err = reader.ReadString('\n')
					if err != nil {
						t.Fatal(err)
					}
					s = strings.TrimSpace(s)
					t.Log(s)

					if s == "DATA" {
						break
					}
				}

				writer.WriteString("354 OK send data ending with <CRLF>.<CRLF>\r\n")
				writer.Flush()
				data := []byte{}
				for {
					d, err := reader.ReadSlice('\n')
					if err != nil {
						t.Fatal(err)
					}
					if d[0] == 46 && d[1] == 13 && d[2] == 10 {
						break
					}
					data = append(data, d...)
				}

				writer.WriteString("250 Server has transmitted the message\n\r")
				writer.Flush()

				m, err := mail.ReadMessage(bytes.NewReader(data))
				if err != nil {
					log.Fatal(err)
				}

				t.Log("Date:", m.Header.Get("Date"))
				t.Log("From:", m.Header.Get("From"))
				t.Log("To:", m.Header.Get("To"))
				t.Log("Subject:", m.Header.Get("Subject"))

				body, err := ioutil.ReadAll(m.Body)
				if err != nil {
					log.Fatal(err)
				}
				t.Logf("%s", body)

				message := SMTPMessage{}
				from, err := m.Header.AddressList("From")
				if err != nil {
					log.Fatal(err)
				}
				if len(from) > 0 {
					message.From = from[0]
				}
				message.To, err = m.Header.AddressList("To")
				if err != nil {
					log.Fatal(err)
				}
				message.Subject = m.Header.Get("Subject")
				message.Body = string(body)

				recorder.Message = &message
			}(conn)
		}
	}()

	return recorder, nil
}

type SMTPMessage struct {
	From    *mail.Address
	To      []*mail.Address
	Subject string
	Body    string
}

func TestService(t *testing.T) {
	recorder, err := NewSMTPRecorder(t)
	if err != nil {
		t.Fatalf("smtp listen: %s", err)
	}

	from := `"Gopher" <gopher@gopherpit.com>`
	defaultFrom := `noreply@gopherpit.com`
	to := []string{`"GopherPit Support" <support@gopherpit.com>`, "contact@gopherpit.com"}
	notifyTo := []string{`"GopherPit Operations" <operations@gopherpit.com>`}
	subject := "test subject"
	body := "test body"

	service := Service{
		SMTPHost:        "localhost",
		SMTPPort:        recorder.Port,
		SMTPSkipVerify:  true,
		NotifyAddresses: notifyTo,
		DefaultFrom:     defaultFrom,
	}

	t.Run("SendEmail", func(t *testing.T) {
		if err := service.SendEmail(from, to, subject, body); err != nil {
			t.Errorf("send email: %s", err)
		}

		recordedFrom := recorder.Message.From.String()
		if recordedFrom != from && recordedFrom != "<"+defaultFrom+">" {
			t.Errorf("message from: expected %s, got %s", from, recordedFrom)
		}

		for _, pt := range to {
			found := false
			for _, rt := range recorder.Message.To {
				if pt == rt.String() || "<"+pt+">" == rt.String() {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("recipient not found %s", pt)
			}
		}

		recordedSubject := recorder.Message.Subject
		if recordedSubject != subject {
			t.Errorf(`message subject: expected "%s", got "%s"`, subject, recordedSubject)
		}

		recordedBody := recorder.Message.Body
		if recordedBody != body+"\r\n" {
			t.Errorf(`message body: expected "%v", got "%v"`, body, recordedBody)
		}
	})

	t.Run("Notify", func(t *testing.T) {
		if err := service.Notify(subject, body); err != nil {
			t.Errorf("send email: %s", err)
		}

		recordedFrom := recorder.Message.From.String()
		if recordedFrom != defaultFrom && recordedFrom != "<"+defaultFrom+">" {
			t.Errorf("message from: expected %s, got %s", defaultFrom, recordedFrom)
		}

		for _, pt := range notifyTo {
			found := false
			for _, rt := range recorder.Message.To {
				if pt == rt.String() || "<"+pt+">" == rt.String() {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("recipient not found %s", pt)
			}
		}

		recordedSubject := recorder.Message.Subject
		if recordedSubject != subject {
			t.Errorf(`message subject: expected "%s", got "%s"`, subject, recordedSubject)
		}

		recordedBody := recorder.Message.Body
		if recordedBody != body+"\r\n" {
			t.Errorf(`message body: expected "%v", got "%v"`, body, recordedBody)
		}
	})

	t.Run("NotifyNoOp", func(t *testing.T) {
		recorder.Message = nil
		service.NotifyAddresses = nil
		if err := service.Notify(subject, body); err != nil {
			t.Errorf("send email: %s", err)
		}
		if recorder.Message != nil {
			t.Errorf("expected no-op, but message %#v has been recorded", recorder.Message)
		}
	})
}
