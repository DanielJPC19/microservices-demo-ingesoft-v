package com.ingesoft.vote;

import com.ingesoft.vote.controller.VoteController;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.test.web.servlet.MockMvc;

import java.util.concurrent.CompletableFuture;

import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@WebMvcTest(VoteController.class)
class VoteControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private KafkaTemplate<String, String> kafkaTemplate;

    @Test
    void getIndexReturnsVotePageWithOptions() throws Exception {
        mockMvc.perform(get("/"))
                .andExpect(status().isOk())
                .andExpect(view().name("index"))
                .andExpect(model().attributeExists("optionA"))
                .andExpect(model().attributeExists("optionB"))
                .andExpect(model().attributeExists("hostname"))
                .andExpect(model().attribute("vote", (Object) null));
    }

    @Test
    void getIndexSetsCookieWhenNonePresent() throws Exception {
        mockMvc.perform(get("/"))
                .andExpect(status().isOk())
                .andExpect(cookie().exists("voter_id"));
    }

    @Test
    void getIndexPreservesExistingVoterIdCookie() throws Exception {
        mockMvc.perform(get("/").cookie(new jakarta.servlet.http.Cookie("voter_id", "test-voter-123")))
                .andExpect(status().isOk())
                .andExpect(cookie().value("voter_id", "test-voter-123"));
    }

    @Test
    void postVoteSendsMessageToKafka() throws Exception {
        when(kafkaTemplate.send(anyString(), anyString(), anyString()))
                .thenReturn(CompletableFuture.completedFuture(null));

        mockMvc.perform(post("/")
                        .cookie(new jakarta.servlet.http.Cookie("voter_id", "voter-abc"))
                        .param("vote", "a"))
                .andExpect(status().isOk())
                .andExpect(view().name("index"))
                .andExpect(model().attribute("vote", "a"));

        verify(kafkaTemplate).send(eq("votes"), eq("voter-abc"), eq("a"));
    }

    @Test
    void postVoteWithOptionBSendsCorrectMessage() throws Exception {
        when(kafkaTemplate.send(anyString(), anyString(), anyString()))
                .thenReturn(CompletableFuture.completedFuture(null));

        mockMvc.perform(post("/")
                        .cookie(new jakarta.servlet.http.Cookie("voter_id", "voter-xyz"))
                        .param("vote", "b"))
                .andExpect(status().isOk())
                .andExpect(model().attribute("vote", "b"));

        verify(kafkaTemplate).send(eq("votes"), eq("voter-xyz"), eq("b"));
    }

    @Test
    void postVoteWithoutCookieGeneratesNewVoterId() throws Exception {
        when(kafkaTemplate.send(anyString(), anyString(), anyString()))
                .thenReturn(CompletableFuture.completedFuture(null));

        ArgumentCaptor<String> voterCaptor = ArgumentCaptor.forClass(String.class);

        mockMvc.perform(post("/").param("vote", "a"))
                .andExpect(status().isOk())
                .andExpect(cookie().exists("voter_id"));

        verify(kafkaTemplate).send(eq("votes"), voterCaptor.capture(), eq("a"));
        // A UUID was generated — just verify it is non-empty
        assert !voterCaptor.getValue().isBlank();
    }
}
