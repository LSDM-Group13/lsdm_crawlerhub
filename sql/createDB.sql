create database LSDM_Group_Project;
use LSDM_Group_Project;

CREATE TABLE Host (
                      HostID INT AUTO_INCREMENT PRIMARY KEY,
                      DomainName VARCHAR(255) NOT NULL,
                      LastCrawledDate DATE
);

CREATE TABLE WebPage (
                         WebPageID INT AUTO_INCREMENT PRIMARY KEY,
                         HostID INT,
                         WebPageURL VARCHAR(255) NOT NULL,
                         Data TEXT,
                         FOREIGN KEY (HostID) REFERENCES Host(HostID)
);

